package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestAQuickerPaceBringsInABoxDueLaterThanItsLongestWait(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	rulesOf(registry).Boxes = Pace{MinInterval: 10 * time.Second, MaxInterval: 10 * time.Second, MaxChargesPerHour: 6}
	registry.sweep(t.Context())
	clock.Advance(11 * time.Second)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())

	assert.NotNil(t, offered(t, events), "due a minute after the first click, now ten seconds after the pace changed")
}

func TestASlowerPaceKeepsTheBoxAlreadyDueAndSpacesTheNext(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	rulesOf(registry).Boxes = Pace{MinInterval: 5 * time.Minute, MaxInterval: 5 * time.Minute, MaxChargesPerHour: 6}
	waitOut(t, registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)
	_, claimed := registry.Claim(offer.Token, "scope-a", "scope-a")
	require.True(t, claimed)

	waitOut(t, registry, clock)
	assert.Nil(t, offered(t, events), "the next wait is drawn at the new pace")

	clock.Advance(5 * time.Minute)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.NotNil(t, offered(t, events))
}

func TestThePaceSetsTheHourlyCap(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")
	rulesOf(registry).Boxes.MaxChargesPerHour = 8

	for range 8 {
		waitOut(t, registry, clock)
		clicked(registry, "scope-a")
		registry.sweep(t.Context())
		offer := offered(t, events)
		require.NotNil(t, offer)
		_, claimed := registry.Claim(offer.Token, "scope-a", "scope-a")
		require.True(t, claimed)
	}

	waitOut(t, registry, clock)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.Nil(t, offered(t, events), "eight, not the config's six")
}

func TestNoBoxIsOfferedBeforeARuleSetIsInForce(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	registry := New(Config{}, clock, NewMemoryStore(), func() Rules { return Rules{} }, fixedBank{})
	events := playing(t, registry, "scope-a")

	for range 3 {
		clock.Advance(time.Minute)
		clicked(registry, "scope-a")
		registry.sweep(t.Context())
	}

	assert.Nil(t, offered(t, events), "a cap of 0 an hour offers nothing")
}

func TestAQuickerQuizPaceBringsInAQuizDueLaterThanItsLongestWait(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")

	rulesOf(registry).Quiz.MinInterval = 10 * time.Second
	rulesOf(registry).Quiz.MaxInterval = 10 * time.Second
	registry.sweep(t.Context())
	clock.Advance(11 * time.Second)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())

	assert.NotNil(t, quizOffered(t, events), "due three minutes after the first click, now ten seconds after the pace changed")
}

func TestQuizzesSwitchedOnAskEachCallerAWholeWaitLater(t *testing.T) {
	registry, clock := newRuledRegistry(func(rules *Rules) {
		rules.Quiz = quizzing(quizWindow)
		rules.Quiz.Enabled = false
	})
	events := playing(t, registry, "scope-a")
	clock.Advance(time.Hour)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())

	rulesOf(registry).Quiz.Enabled = true
	registry.sweep(t.Context())
	assert.Nil(t, quizOffered(t, events), "a wait long past is not a quiz for everyone at once")

	waitOutQuiz(t, registry, clock)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.NotNil(t, quizOffered(t, events))
}

func TestQuizzesSwitchedOffAskNoMoreAndLetTheOneAskedBeAnswered(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := playing(t, registry, "scope-a")
	waitOutQuiz(t, registry, clock)
	offer := quizOffered(t, events)
	require.NotNil(t, offer)

	rulesOf(registry).Quiz.Enabled = false
	registry.sweep(t.Context())
	_, opened := registry.OpenQuiz(offer.Token, "scope-a")
	require.True(t, opened)
	answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", 1)
	require.True(t, ok)
	assert.True(t, answered.Correct)

	clock.Advance(time.Hour)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.Nil(t, quizOffered(t, events))
}

func TestABoxLastsTheOfferTTLInForceWhenItIsOffered(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	rulesOf(registry).OfferTTL = 40 * time.Second
	waitOut(t, registry, clock)

	offer := offered(t, events)
	require.NotNil(t, offer)
	assert.Equal(t, clock.Now().Add(40*time.Second), offer.ExpiresAt)
}
