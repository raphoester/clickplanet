package quizzes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

const (
	quizWindow = 3 * time.Minute
	quizRetry  = 30 * time.Second
	answerIn   = 5 * time.Second
	bannerFor  = 20 * time.Second
)

func quizzing() Rules {
	return Rules{
		Enabled:                true,
		MinInterval:            quizWindow,
		MaxInterval:            quizWindow,
		MissRetry:              quizRetry,
		OfferTTL:               bannerFor,
		AnswerWindow:           answerIn,
		MaxRightAnswersPerHour: 6,
		ActiveWithin:           5 * time.Minute,
	}
}

func newQuizzingRegistry(t *testing.T) (*Registry, *cptime.FixedClock) {
	t.Helper()

	return newRuledRegistry(t, func(*Rules) {})
}

func newRuledRegistry(t *testing.T, change func(*Rules)) (*Registry, *cptime.FixedClock) {
	t.Helper()

	clock := cptime.NewFixedClock(epoch)
	rules := quizzing()
	change(&rules)

	read := &rules
	registry := New(Config{ForgetAfter: 5 * time.Minute, SweepInterval: time.Second}, clock,
		func() Rules { return *read }, fixedBank{})
	testRules[registry] = read
	t.Cleanup(func() { delete(testRules, registry) })

	return registry, clock
}

// The rules each registry of these tests reads, for a test to change.
var testRules = map[*Registry]*Rules{}

func rulesOf(r *Registry) *Rules {
	return testRules[r]
}

func accountOf(entrant Entrant) Account {
	return Account("acc-" + string(entrant))
}

func attend(t *testing.T, r *Registry, entrant Entrant) <-chan Event {
	t.Helper()

	events, leave := r.Attend(entrant)
	t.Cleanup(leave)

	return events
}

func clicked(r *Registry, entrant Entrant) {
	r.Clicked(Click{Caller: Caller{Scope: string(entrant), Account: accountOf(entrant)}})
}

func playing(t *testing.T, r *Registry, entrant Entrant) <-chan Event {
	t.Helper()

	events := attend(t, r, entrant)
	clicked(r, entrant)

	return events
}

type fixedBank struct{}

const (
	rightAnswer = "Tallinn"
	wrongAnswer = "Riga"
)

func (fixedBank) Draw(float64, bank.Fit) bank.Round {
	return bank.Round{
		Question: bank.Question{ID: "capital:ee", Subject: "ee", Text: "What is the capital of Estonia?"},
		Options:  []string{wrongAnswer, rightAnswer, "Vilnius"},
		Correct:  1,
	}
}

// aimingBank keeps the fit each draw was asked for.
type aimingBank struct {
	fixedBank
	fits []bank.Fit
}

func (b *aimingBank) Draw(lean float64, fit bank.Fit) bank.Round {
	b.fits = append(b.fits, fit)
	return b.fixedBank.Draw(lean, fit)
}

func waitOutQuiz(t *testing.T, r *Registry, clock *cptime.FixedClock) {
	t.Helper()

	clock.Advance(quizWindow + time.Second)
	r.sweep()
}

func letTheBannerLapse(t *testing.T, r *Registry, clock *cptime.FixedClock) {
	t.Helper()

	clock.Advance(bannerFor + time.Second)
	r.sweep()
}

func quizOffered(t *testing.T, events <-chan Event) *Offer {
	t.Helper()

	for {
		select {
		case event := <-events:
			if event.Offer != nil {
				return event.Offer
			}
		default:
			return nil
		}
	}
}

func takeQuiz(t *testing.T, r *Registry, clock *cptime.FixedClock, entrant Entrant) (*Offer, Asked) {
	t.Helper()

	events := playing(t, r, entrant)
	waitOutQuiz(t, r, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer, "expected a quiz")

	asked, ok := r.OpenQuiz(offer.Token, entrant)
	require.True(t, ok)

	return offer, asked
}

func indexOf(options []string, want string) int {
	for i, option := range options {
		if option == want {
			return i
		}
	}

	return -1
}

func TestTheDrawIsAskedForTheSetsFit(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	questions := &aimingBank{}
	registry.bank = questions
	fit := bank.Fit{Target: 0.7, Width: 1, Floor: 0.42, FloorAfter: 15, Explore: 0.1}
	rulesOf(registry).Fit = fit

	_ = playing(t, registry, "scope-a")
	waitOutQuiz(t, registry, clock)

	assert.Equal(t, []bank.Fit{fit}, questions.fits)
}

func TestAQuizGoesToAnAttendingCallerOnceItsOwnWindowPasses(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	registry.sweep()
	assert.Nil(t, quizOffered(t, events), "nothing is due yet")

	waitOutQuiz(t, registry, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer)
	assert.Equal(t, clock.Now().Add(bannerFor), offer.ExpiresAt)
}

func TestEveryCallerIsOnItsOwnSchedule(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)

	channels := map[Entrant]<-chan Event{}
	for _, entrant := range []Entrant{"a", "b", "c"} {
		channels[entrant] = playing(t, registry, entrant)
	}

	waitOutQuiz(t, registry, clock)

	for entrant, events := range channels {
		assert.NotNilf(t, quizOffered(t, events), "%s was not offered a quiz on its own turn", entrant)
	}
}

func TestAQuizCarriesNoQuestionUntilItIsOpened(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	assert.Equal(t, Offer{Token: offer.Token, ExpiresAt: offer.ExpiresAt}, *offer,
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

func TestARightAnswerInTimeSaysWhatTheQuestionWasAbout(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn - time.Second)

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
	require.True(t, ok)
	assert.Equal(t, ratings.Right, answered.Outcome)
	assert.Equal(t, "ee", answered.Subject, "the broadcast says what the question was about")
	assert.False(t, answered.Welcome)
}

func TestAWrongAnswerLandsAndSaysTheRightChoice(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, wrongAnswer))
	require.True(t, ok, "a wrong answer is not a failed call")
	assert.Equal(t, ratings.Wrong, answered.Outcome)
	assert.Equal(t, indexOf(asked.Options, rightAnswer), answered.CorrectChoice,
		"it says which one was right, so the player learns something")
}

func TestAChoicePastTheThreeIsThePagesTimeout(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", len(asked.Options))
	require.True(t, ok)
	assert.Equal(t, ratings.TimedOut, answered.Outcome)
}

func TestAnAnswerPastTheDeadlineIsWrong(t *testing.T) {
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

	letTheBannerLapse(t, registry, clock)

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
	registry.sweep()

	assert.NotNil(t, quizOffered(t, events), "a missed quiz should come back sooner")
}

func TestASecondMissedQuizInARowWaitsTheOrdinaryWindow(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	waitOutQuiz(t, registry, clock)
	require.NotNil(t, quizOffered(t, events))
	letTheBannerLapse(t, registry, clock)

	clock.Advance(quizRetry + time.Second)
	registry.sweep()
	require.NotNil(t, quizOffered(t, events))
	letTheBannerLapse(t, registry, clock)

	clock.Advance(quizRetry + time.Second)
	registry.sweep()
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
	registry.sweep()

	clock.Advance(quizRetry + time.Second)
	registry.sweep()

	assert.NotNil(t, quizOffered(t, events))
}

func TestAnsweringClearsTheMissThatCameBefore(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	waitOutQuiz(t, registry, clock)
	require.NotNil(t, quizOffered(t, events))
	letTheBannerLapse(t, registry, clock)

	clock.Advance(quizRetry + time.Second)
	registry.sweep()
	offer := quizOffered(t, events)
	require.NotNil(t, offer)

	_, ok := registry.OpenQuiz(offer.Token, "scope-a")
	require.True(t, ok)

	_, ok = registry.AnswerQuiz(offer.Token, "scope-a", 99)
	require.True(t, ok)

	assert.Equal(t, 0, registry.callers["scope-a"].misses, "a wrong answer is an answer, not a miss")
}

func TestAQuestionOpenedAndLeftLapsesAtItsDeadline(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn + time.Second)
	registry.sweep()

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
	require.True(t, ok, "an answer a moment late is still counted")
	assert.True(t, answered.Late, "the sweep took it: to the player there is nothing left to answer")
	assert.Equal(t, ratings.Wrong, answered.Outcome, "it came past the deadline")
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

func TestALateAnswerIsCountedOnceAndChangesNoSchedule(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn + time.Second)
	registry.sweep()
	missed := *registry.callers["scope-a"]

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", len(asked.Options))
	require.True(t, ok)
	assert.True(t, answered.Late)
	assert.Equal(t, ratings.TimedOut, answered.Outcome, "the page's timeout, which reached the server after the sweep")
	assert.Equal(t, answerIn+time.Second, answered.After)

	assert.Equal(t, 1, registry.callers["scope-a"].misses, "it lapsed: a miss, as before")
	assert.Equal(t, missed.nextAt, registry.callers["scope-a"].nextAt, "the next quiz comes as it would have")

	_, ok = registry.AnswerQuiz(offer.Token, "scope-a", 0)
	assert.False(t, ok, "counted once")
}

func TestAnAnswerLongAfterTheDeadlineIsNotCounted(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, _ := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn + time.Second)
	registry.sweep()
	clock.Advance(lateAnswers)
	registry.sweep()

	_, ok := registry.AnswerQuiz(offer.Token, "scope-a", 0)
	assert.False(t, ok, "a page that comes back this late was left, and a question left is not an answer")
	assert.Empty(t, registry.lapsed, "nothing is kept of it")
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
	registry.sweep()

	_, ok := registry.AnswerQuiz(offer.Token, "scope-b", 3)
	assert.False(t, ok)
}

func TestAQuizGoesOnlyToACallerWhoIsPlaying(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)

	events := attend(t, registry, "scope-a")
	waitOutQuiz(t, registry, clock)

	assert.Nil(t, quizOffered(t, events))
}

func TestACallerWithNoAccountIsAskedNothing(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)

	events := attend(t, registry, "scope-a")
	registry.Clicked(Click{Caller: Caller{Scope: "scope-a"}})
	waitOutQuiz(t, registry, clock)

	assert.Nil(t, quizOffered(t, events), "a right answer of nobody's pays nobody")
}

func TestAClickHeardLongAfterItWasTriedIsNoSignOfAPlayer(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := attend(t, registry, "scope-a")

	registry.Clicked(Click{
		Caller: Caller{Scope: "scope-a", Account: accountOf("scope-a")},
		At:     clock.Now().Add(-6 * time.Minute),
	})
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
	assert.Nil(t, quizOffered(t, events), "six right answers an hour, and no more")

	clock.Advance(time.Hour)
	clicked(registry, "scope-a")
	registry.sweep()
	assert.NotNil(t, quizOffered(t, events))
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
	registry.sweep()

	_, ok := registry.OpenQuiz(offer.Token, "scope-a")
	assert.False(t, ok, "a token left behind would be answerable by whoever next got this scope")
}

func TestAQuizWonIsToldToEveryStream(t *testing.T) {
	registry, _ := newQuizzingRegistry(t)
	mine := attend(t, registry, "scope-a")
	theirs := attend(t, registry, "scope-b")

	registry.Publish(Won{CountryID: "fr", Subject: "ee"})

	for _, events := range []<-chan Event{mine, theirs} {
		event := <-events
		require.NotNil(t, event.Won)
		assert.Equal(t, Won{CountryID: "fr", Subject: "ee"}, *event.Won)
	}
}

func TestEveryOfferIsReported(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	var offered, lapsed []string
	var answered []bool
	registry.Observe(Report{
		Offered:  func() { offered = append(offered, "one") },
		Lapsed:   func(scope string) { lapsed = append(lapsed, scope) },
		Answered: func(_ string, correct bool, _ time.Duration) { answered = append(answered, correct) },
	})

	offer, asked := takeQuiz(t, registry, clock, "scope-a")
	_, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
	require.True(t, ok)

	events := playing(t, registry, "scope-b")
	waitOutQuiz(t, registry, clock)
	require.NotNil(t, quizOffered(t, events))
	letTheBannerLapse(t, registry, clock)

	assert.Len(t, offered, 2)
	assert.Equal(t, []bool{true}, answered)
	assert.Equal(t, []string{"scope-b"}, lapsed)
}
