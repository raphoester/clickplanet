package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	quizWindow = 3 * time.Minute
	quizRetry  = 30 * time.Second
	answerIn   = 5 * time.Second
	bannerFor  = 20 * time.Second
)

func newQuizzingRegistry(t *testing.T) (*Registry, *cptime.FixedClock) {
	t.Helper()

	return newRuledRegistry(func(rules *Rules) { rules.Quiz = quizzing(quizWindow) })
}

func quizzing(every time.Duration) QuizRules {
	return QuizRules{
		Enabled:           true,
		MinInterval:       every,
		MaxInterval:       every,
		MissRetry:         quizRetry,
		OfferTTL:          bannerFor,
		AnswerWindow:      answerIn,
		MaxChargesPerHour: 6,
	}
}

type fixedBank struct{}

const (
	rightAnswer = "Tallinn"
	wrongAnswer = "Riga"
)

func (fixedBank) Draw(float64, quizzes.Fit) quizzes.Round {
	return quizzes.Round{
		Question: quizzes.Question{ID: "capital:ee", Subject: "ee", Text: "What is the capital of Estonia?"},
		Options:  []string{wrongAnswer, rightAnswer, "Vilnius"},
		Correct:  1,
	}
}

// aimingBank keeps the fit each draw was asked for.
type aimingBank struct {
	fixedBank
	fits []quizzes.Fit
}

func (b *aimingBank) Draw(lean float64, fit quizzes.Fit) quizzes.Round {
	b.fits = append(b.fits, fit)
	return b.fixedBank.Draw(lean, fit)
}

func TestTheDrawIsAskedForTheSetsFit(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	bank := &aimingBank{}
	registry.bank = bank
	fit := quizzes.Fit{Target: 0.7, Width: 1, Floor: 0.42, FloorAfter: 15, Explore: 0.1}
	rulesOf(registry).Quiz.Fit = fit

	_ = playing(t, registry, "scope-a")
	waitOutQuiz(t, registry, clock)

	assert.Equal(t, []quizzes.Fit{fit}, bank.fits)
}

func waitOutQuiz(t *testing.T, r *Registry, clock *cptime.FixedClock) {
	t.Helper()

	clock.Advance(quizWindow + time.Second)
	r.sweep(t.Context())
}

func letTheBannerLapse(t *testing.T, r *Registry, clock *cptime.FixedClock) {
	t.Helper()

	clock.Advance(bannerFor + time.Second)
	r.sweep(t.Context())
}

func quizOffered(t *testing.T, events <-chan Event) *QuizOffer {
	t.Helper()

	for {
		select {
		case event := <-events:
			if event.Quiz != nil {
				return event.Quiz
			}
		default:
			return nil
		}
	}
}

func takeQuiz(t *testing.T, r *Registry, clock *cptime.FixedClock, entrant Entrant) (*QuizOffer, Asked) {
	t.Helper()

	events := playing(t, r, entrant)
	waitOutQuiz(t, r, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer, "expected a quiz")

	asked, ok := r.OpenQuiz(offer.Token, entrant)
	require.True(t, ok)

	return offer, asked
}

func TestAQuizGoesToAnAttendingCallerOnceItsOwnWindowPasses(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	registry.sweep(t.Context())
	assert.Nil(t, quizOffered(t, events), "nothing is due yet")

	waitOutQuiz(t, registry, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer)
	assert.Equal(t, clock.Now().Add(bannerFor), offer.ExpiresAt)
}

func TestAQuizCarriesNoQuestionUntilItIsOpened(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	assert.Equal(t, QuizOffer{Token: offer.Token, ExpiresAt: offer.ExpiresAt}, *offer,
		"the banner carries a token and a deadline, and nothing else at all")
	assert.Equal(t, "What is the capital of Estonia?", asked.Question)
	assert.Len(t, asked.Options, 3)
	assert.Equal(t, answerIn, asked.Window)
	assert.Equal(t, clock.Now().Add(answerIn), asked.Deadline)
}

func TestOpeningAQuizTwiceIsTheSameQuestionAndTheSameDeadline(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, first := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(2 * time.Second)

	second, ok := registry.OpenQuiz(offer.Token, "scope-a")
	require.True(t, ok)

	assert.Equal(t, first.Question, second.Question)
	assert.Equal(t, first.Options, second.Options)
	assert.Equal(t, first.Deadline, second.Deadline, "a reload is not a second window")
}

func TestARightAnswerInTimeIsWorthACharge(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn - time.Second)

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
	require.True(t, ok)
	assert.Equal(t, ratings.Right, answered.Outcome)
	assert.Equal(t, KindRefill, answered.Reward.Kind)
	assert.Equal(t, "ee", answered.Subject, "the broadcast says what the question was about")
}

func TestAWrongAnswerLandsAndIsWorthNothing(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, wrongAnswer))
	require.True(t, ok, "a wrong answer is not a failed call")
	assert.Equal(t, ratings.Wrong, answered.Outcome)
	assert.Empty(t, answered.Reward.Kind)
	assert.Equal(t, indexOf(asked.Options, rightAnswer), answered.CorrectChoice,
		"it says which one was right, so the player learns something")
}

func TestAChoicePastTheEndIsWrongRatherThanAFault(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, _ := takeQuiz(t, registry, clock, "scope-a")

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", 99)
	require.True(t, ok)
	assert.Equal(t, ratings.TimedOut, answered.Outcome)
}

func TestAnAnswerPastTheDeadlineIsWorthNothing(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn + time.Millisecond)

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
	require.True(t, ok)
	assert.Equal(t, ratings.Wrong, answered.Outcome, "the deadline is the server's own stamp, not the client's countdown")
}

func TestAQuizCannotBeAnsweredWithoutBeingRead(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")
	waitOutQuiz(t, registry, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer)

	_, ok := registry.AnswerQuiz(offer.Token, "scope-a", 0)
	assert.False(t, ok)
}

func TestAQuizIsOnlyItsOwnCallersToOpenOrAnswer(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	_, ok := registry.OpenQuiz(offer.Token, "scope-b")
	assert.False(t, ok)

	_, ok = registry.AnswerQuiz(offer.Token, "scope-b", indexOf(asked.Options, rightAnswer))
	assert.False(t, ok)
}

func TestAQuizIsAnsweredOnce(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	_, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
	require.True(t, ok)

	_, ok = registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
	assert.False(t, ok, "the token is spent, right answer or not")
}

func TestABannerNobodyOpensLapsesAndFreesTheSlot(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")
	waitOutQuiz(t, registry, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer)

	clock.Advance(bannerFor + time.Second)
	registry.sweep(t.Context())

	_, ok := registry.OpenQuiz(offer.Token, "scope-a")
	assert.False(t, ok)

	clicked(registry, "scope-a")
	waitOutQuiz(t, registry, clock)
	assert.NotNil(t, quizOffered(t, events), "the caller is due another once its window passes")
}

func TestAMissedQuizBringsTheNextOneForwardOnce(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	waitOutQuiz(t, registry, clock)
	require.NotNil(t, quizOffered(t, events))

	letTheBannerLapse(t, registry, clock)
	require.Nil(t, quizOffered(t, events))

	clock.Advance(quizRetry + time.Second)
	registry.sweep(t.Context())

	assert.NotNil(t, quizOffered(t, events), "a missed quiz should come back sooner")
}

func TestASecondMissedQuizInARowWaitsTheOrdinaryWindow(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	waitOutQuiz(t, registry, clock)
	require.NotNil(t, quizOffered(t, events))
	letTheBannerLapse(t, registry, clock)

	clock.Advance(quizRetry + time.Second)
	registry.sweep(t.Context())
	require.NotNil(t, quizOffered(t, events))
	letTheBannerLapse(t, registry, clock)

	clock.Advance(quizRetry + time.Second)
	registry.sweep(t.Context())
	assert.Nil(t, quizOffered(t, events), "the second miss is not brought forward")

	clicked(registry, "scope-a")
	waitOutQuiz(t, registry, clock)
	assert.NotNil(t, quizOffered(t, events))
}

func TestAQuestionOpenedAndLeftCountsAsAMiss(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	waitOutQuiz(t, registry, clock)
	offer := quizOffered(t, events)
	require.NotNil(t, offer)

	_, ok := registry.OpenQuiz(offer.Token, "scope-a")
	require.True(t, ok)

	clock.Advance(answerIn + time.Second)
	registry.sweep(t.Context())

	clock.Advance(quizRetry + time.Second)
	registry.sweep(t.Context())

	assert.NotNil(t, quizOffered(t, events))
}

func TestAnsweringClearsTheQuizMissThatCameBefore(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	waitOutQuiz(t, registry, clock)
	require.NotNil(t, quizOffered(t, events))
	letTheBannerLapse(t, registry, clock)

	clock.Advance(quizRetry + time.Second)
	registry.sweep(t.Context())
	offer := quizOffered(t, events)
	require.NotNil(t, offer)

	_, ok := registry.OpenQuiz(offer.Token, "scope-a")
	require.True(t, ok)

	_, ok = registry.AnswerQuiz(offer.Token, "scope-a", 99)
	require.True(t, ok)

	assert.Equal(t, 0, registry.callers["scope-a"].quizMisses, "a wrong answer is an answer, not a miss")
}

func TestAQuestionOpenedAndLeftLapsesAtItsDeadline(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn + time.Second)
	registry.sweep(t.Context())

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
	require.True(t, ok, "an answer a moment late is still counted")
	assert.True(t, answered.Late, "the sweep took it: to the player there is nothing left to answer")
	assert.Equal(t, ratings.Wrong, answered.Outcome, "it came past the deadline")
	assert.Zero(t, answered.Reward)
}

func TestAnAnswerSaysItsQuestionAndHowLongAfterTheOpeningItLanded(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(2 * time.Second)

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, wrongAnswer))
	require.True(t, ok)
	assert.Equal(t, ratings.QuestionID("capital:ee"), answered.Question)
	assert.Equal(t, 2*time.Second, answered.After, "from the opening, not from the banner")
	assert.Equal(t, ratings.Wrong, answered.Outcome)
	assert.False(t, answered.Late)
}

func TestAChoicePastTheThreeIsThePagesTimeout(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", len(asked.Options))
	require.True(t, ok)
	assert.Equal(t, ratings.TimedOut, answered.Outcome)
}

func TestALateAnswerIsCountedOnceAndChangesNoSchedule(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn + time.Second)
	registry.sweep(t.Context())
	missed := *registry.callers["scope-a"]

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", len(asked.Options))
	require.True(t, ok)
	assert.True(t, answered.Late)
	assert.Equal(t, ratings.TimedOut, answered.Outcome, "the page's timeout, which reached the server after the sweep")
	assert.Equal(t, answerIn+time.Second, answered.After)

	assert.Equal(t, 1, registry.callers["scope-a"].quizMisses, "it lapsed: a miss, as before")
	assert.Equal(t, missed.nextQuizAt, registry.callers["scope-a"].nextQuizAt, "the next quiz comes as it would have")

	_, ok = registry.AnswerQuiz(offer.Token, "scope-a", 0)
	assert.False(t, ok, "counted once")
}

func TestAnAnswerLongAfterTheDeadlineIsNotCounted(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, _ := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn + time.Second)
	registry.sweep(t.Context())
	clock.Advance(lateAnswers)
	registry.sweep(t.Context())

	_, ok := registry.AnswerQuiz(offer.Token, "scope-a", 0)
	assert.False(t, ok, "a page that comes back this late was left, and a question left is not an answer")
	assert.Empty(t, registry.lapsedQuizzes, "nothing is kept of it")
}

func TestABannerNeverOpenedIsNeverAnswered(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")
	waitOutQuiz(t, registry, clock)
	offer := quizOffered(t, events)
	require.NotNil(t, offer)

	letTheBannerLapse(t, registry, clock)

	_, ok := registry.AnswerQuiz(offer.Token, "scope-a", 3)
	assert.False(t, ok, "the player never saw the question")
}

func TestALateAnswerIsOnlyItsOwnCallers(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, _ := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn + time.Second)
	registry.sweep(t.Context())

	_, ok := registry.AnswerQuiz(offer.Token, "scope-b", 3)
	assert.False(t, ok)
}

func TestQuizzesAndBoxesRunOnSeparateClocks(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	require.NotNil(t, offered(t, events), "the box is due")

	drain(events)
	clicked(registry, "scope-a")
	waitOutQuiz(t, registry, clock)

	assert.NotNil(t, quizOffered(t, events), "and the quiz on its own clock, not instead of a box")
}

func TestAQuizIsNotOfferedForAKindTheCallerIsAlreadyHolding(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	roomOf(registry).Keep(accountOf("scope-a"), KindRefill)

	waitOutQuiz(t, registry, clock)
	assert.Nil(t, quizOffered(t, events), "there is nothing a right answer could be worth")

	roomOf(registry).Keep(accountOf("scope-a"))
	clicked(registry, "scope-a")
	waitOutQuiz(t, registry, clock)
	assert.NotNil(t, quizOffered(t, events))
}

func TestAQuizGoesOnlyToACallerWhoIsPlaying(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)

	events := attend(t, registry, "scope-a")
	waitOutQuiz(t, registry, clock)

	assert.Nil(t, quizOffered(t, events))
}

func TestQuizzesStopAtTheirOwnHourlyCap(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	for range 6 {
		clicked(registry, "scope-a")
		waitOutQuiz(t, registry, clock)

		offer := quizOffered(t, events)
		require.NotNil(t, offer)

		asked, ok := registry.OpenQuiz(offer.Token, "scope-a")
		require.True(t, ok)

		answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
		require.True(t, ok)
		require.Equal(t, ratings.Right, answered.Outcome)
	}

	clicked(registry, "scope-a")
	waitOutQuiz(t, registry, clock)
	assert.Nil(t, quizOffered(t, events), "six an hour, and no more")

	clock.Advance(time.Hour)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.NotNil(t, quizOffered(t, events))
}

func TestNoQuizIsOfferedWithoutABank(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	for range 10 {
		clicked(registry, "scope-a")
		waitOutQuiz(t, registry, clock)
	}

	for {
		select {
		case event := <-events:
			require.Nil(t, event.Quiz)
		default:
			return
		}
	}
}

func TestForgettingACallerTakesItsQuizWithIt(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)

	events, leave := registry.Attend("scope-a")
	clicked(registry, "scope-a")
	waitOutQuiz(t, registry, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer)
	leave()

	clock.Advance(6 * time.Minute)
	registry.sweep(t.Context())

	_, ok := registry.OpenQuiz(offer.Token, "scope-a")
	assert.False(t, ok, "a token left behind would be answerable by whoever next got this scope")
}

func indexOf(options []string, want string) int {
	for i, option := range options {
		if option == want {
			return i
		}
	}

	return -1
}
