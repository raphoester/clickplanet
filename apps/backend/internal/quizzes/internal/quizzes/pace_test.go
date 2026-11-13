package quizzes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAQuickerPaceBringsInAQuizDueLaterThanItsLongestWait(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	rulesOf(registry).MinInterval = 10 * time.Second
	rulesOf(registry).MaxInterval = 10 * time.Second
	registry.sweep()

	clock.Advance(11 * time.Second)
	registry.sweep()
	assert.NotNil(t, quizOffered(t, events), "due three minutes after the first click, now ten seconds after the pace changed")
}

func TestQuizzesSwitchedOnAskEachCallerAWholeWaitLater(t *testing.T) {
	registry, clock := newRuledRegistry(t, func(rules *Rules) { rules.Enabled = false })
	events := playing(t, registry, "scope-a")

	waitOutQuiz(t, registry, clock)
	waitOutQuiz(t, registry, clock)
	clicked(registry, "scope-a")
	rulesOf(registry).Enabled = true
	registry.sweep()
	assert.Nil(t, quizOffered(t, events), "a wait long past is not a quiz for everyone at once")

	clicked(registry, "scope-a")
	waitOutQuiz(t, registry, clock)
	assert.NotNil(t, quizOffered(t, events))
}

func TestQuizzesSwitchedOffAskNoMoreAndLetTheOneAskedBeAnswered(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")
	waitOutQuiz(t, registry, clock)
	offer := quizOffered(t, events)
	require.NotNil(t, offer)

	rulesOf(registry).Enabled = false

	_, opened := registry.OpenQuiz(offer.Token, "scope-a")
	require.True(t, opened)
	_, ok := registry.AnswerQuiz(offer.Token, "scope-a", 1)
	assert.True(t, ok)

	clicked(registry, "scope-a")
	waitOutQuiz(t, registry, clock)
	assert.Nil(t, quizOffered(t, events))
}

func TestAFrozenSetOffersNothingAndLosesTheSlot(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	rulesOf(registry).Frozen = true
	waitOutQuiz(t, registry, clock)
	assert.Nil(t, quizOffered(t, events))

	rulesOf(registry).Frozen = false
	clicked(registry, "scope-a")
	registry.sweep()
	assert.Nil(t, quizOffered(t, events), "the slot was lost, as an away player's: the next one is a wait away")

	waitOutQuiz(t, registry, clock)
	assert.NotNil(t, quizOffered(t, events))
}

func TestABannerLastsTheOfferTTLInForceWhenItIsOffered(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")
	rulesOf(registry).OfferTTL = time.Minute

	waitOutQuiz(t, registry, clock)

	offer := quizOffered(t, events)
	require.NotNil(t, offer)
	assert.Equal(t, clock.Now().Add(time.Minute), offer.ExpiresAt)
}
