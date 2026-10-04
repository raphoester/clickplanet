package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	quizWindow = 3 * time.Minute
	answerIn   = 5 * time.Second
	bannerFor  = 20 * time.Second
)

func newQuizzingRegistry(t *testing.T) (*Registry, *cptime.FixedClock) {
	t.Helper()

	registry, clock := newTestRegistry()
	registry.Quizzing(quizzes.Config{
		Enabled:           true,
		MinInterval:       quizWindow,
		MaxInterval:       quizWindow,
		OfferTTL:          bannerFor,
		AnswerWindow:      answerIn,
		MaxChargesPerHour: 6,
	}, fixedBank{})

	return registry, clock
}

type fixedBank struct{}

const (
	rightAnswer = "Tallinn"
	wrongAnswer = "Riga"
)

func (fixedBank) Draw() quizzes.Round {
	return quizzes.Round{
		Question: quizzes.Question{ID: "capital:ee", Subject: "ee", Text: "What is the capital of Estonia?"},
		Options:  []string{wrongAnswer, rightAnswer, "Vilnius"},
		Correct:  1,
	}
}

func waitOutQuiz(r *Registry, clock *cptime.FixedClock) {
	clock.Advance(quizWindow + time.Second)
	r.sweep()
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
	waitOutQuiz(r, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer, "expected a quiz")

	asked, ok := r.OpenQuiz(offer.Token, entrant)
	require.True(t, ok)

	return offer, asked
}

func TestAQuizGoesToAnAttendingCallerOnceItsOwnWindowPasses(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	registry.sweep()
	assert.Nil(t, quizOffered(t, events), "nothing is due yet")

	waitOutQuiz(registry, clock)

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
	assert.True(t, answered.Correct)
	assert.Equal(t, KindRefill, answered.Reward.Kind)
	assert.Equal(t, "ee", answered.Subject, "the broadcast says what the question was about")
}

func TestAWrongAnswerLandsAndIsWorthNothing(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, wrongAnswer))
	require.True(t, ok, "a wrong answer is not a failed call")
	assert.False(t, answered.Correct)
	assert.Empty(t, answered.Reward.Kind)
	assert.Equal(t, indexOf(asked.Options, rightAnswer), answered.CorrectChoice,
		"it says which one was right, so the player learns something")
}

func TestAChoicePastTheEndIsWrongRatherThanAFault(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, _ := takeQuiz(t, registry, clock, "scope-a")

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", 99)
	require.True(t, ok)
	assert.False(t, answered.Correct)
}

func TestAnAnswerPastTheDeadlineIsWorthNothing(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn + time.Millisecond)

	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
	require.True(t, ok)
	assert.False(t, answered.Correct, "the deadline is the server's own stamp, not the client's countdown")
}

func TestAQuizCannotBeAnsweredWithoutBeingRead(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")
	waitOutQuiz(registry, clock)

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
	waitOutQuiz(registry, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer)

	clock.Advance(bannerFor + time.Second)
	registry.sweep()

	_, ok := registry.OpenQuiz(offer.Token, "scope-a")
	assert.False(t, ok)

	clicked(registry, "scope-a")
	waitOutQuiz(registry, clock)
	assert.NotNil(t, quizOffered(t, events), "the caller is due another once its window passes")
}

func TestAQuestionOpenedAndLeftLapsesAtItsDeadline(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	offer, asked := takeQuiz(t, registry, clock, "scope-a")

	clock.Advance(answerIn + time.Second)
	registry.sweep()

	_, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
	assert.False(t, ok, "the sweep took it: there is nothing left to answer")
}

func TestQuizzesAndBoxesRunOnSeparateClocks(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	require.NotNil(t, offered(t, events), "the box is due")

	drain(events)
	clicked(registry, "scope-a")
	waitOutQuiz(registry, clock)

	assert.NotNil(t, quizOffered(t, events), "and the quiz on its own clock, not instead of a box")
}

func TestAQuizIsNotOfferedForAKindTheCallerIsAlreadyHolding(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	holdingsOf(registry).grant(holderOf("scope-a"), KindRefill)

	waitOutQuiz(registry, clock)
	assert.Nil(t, quizOffered(t, events), "there is nothing a right answer could be worth")

	holdingsOf(registry).take(holderOf("scope-a"))
	clicked(registry, "scope-a")
	waitOutQuiz(registry, clock)
	assert.NotNil(t, quizOffered(t, events))
}

func TestAQuizGoesOnlyToACallerWhoIsPlaying(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)

	events := attend(t, registry, "scope-a")
	waitOutQuiz(registry, clock)

	assert.Nil(t, quizOffered(t, events))
}

func TestQuizzesStopAtTheirOwnHourlyCap(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	for range 6 {
		clicked(registry, "scope-a")
		waitOutQuiz(registry, clock)

		offer := quizOffered(t, events)
		require.NotNil(t, offer)

		asked, ok := registry.OpenQuiz(offer.Token, "scope-a")
		require.True(t, ok)

		answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
		require.True(t, ok)
		require.True(t, answered.Correct)
	}

	clicked(registry, "scope-a")
	waitOutQuiz(registry, clock)
	assert.Nil(t, quizOffered(t, events), "six an hour, and no more")

	clock.Advance(time.Hour)
	clicked(registry, "scope-a")
	registry.sweep()
	assert.NotNil(t, quizOffered(t, events))
}

func TestNoQuizIsOfferedWithoutABank(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	for range 10 {
		clicked(registry, "scope-a")
		waitOutQuiz(registry, clock)
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
	waitOutQuiz(registry, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer)
	leave()

	clock.Advance(6 * time.Minute)
	registry.sweep()

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
