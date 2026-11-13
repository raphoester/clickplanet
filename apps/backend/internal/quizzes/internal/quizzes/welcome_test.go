package quizzes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const welcomeWithin = 10 * time.Minute

func newWelcomingRegistry(t *testing.T) (*Registry, *cptime.FixedClock) {
	t.Helper()

	return newRuledRegistry(t, func(rules *Rules) { rules.Welcome = WelcomeRules{Within: welcomeWithin} })
}

// newcomer is an account made a moment ago, clicking from scope-n.
func newcomer(clock *cptime.FixedClock) Click {
	return Click{
		Caller: Caller{Scope: "scope-n", Account: accountOf("scope-n")},
		Born:   clock.Now().Add(-time.Minute),
	}
}

func dry(click Click) Click {
	click.Dry = true
	return click
}

func TestANewcomerWhoseClickEmptiesItsBankIsAskedAQuizAtOnce(t *testing.T) {
	registry, clock := newWelcomingRegistry(t)
	events := attend(t, registry, "scope-n")

	registry.Clicked(newcomer(clock))
	registry.sweep()
	require.Nil(t, quizOffered(t, events), "clicks still in the bank")

	registry.Clicked(dry(newcomer(clock)))
	registry.sweep()
	offer := quizOffered(t, events)
	require.NotNil(t, offer)

	_, ok := registry.OpenQuiz(offer.Token, "scope-n")
	require.True(t, ok)
	answered, ok := registry.AnswerQuiz(offer.Token, "scope-n", 1)
	require.True(t, ok)
	assert.True(t, answered.Welcome, "the bonuses module pays the welcome its own way")
}

func TestOnlyTheFirstEmptyBankAsksTheWelcomeQuiz(t *testing.T) {
	registry, clock := newWelcomingRegistry(t)
	events := attend(t, registry, "scope-n")

	registry.Clicked(dry(newcomer(clock)))
	registry.sweep()
	offer := quizOffered(t, events)
	require.NotNil(t, offer)
	_, ok := registry.OpenQuiz(offer.Token, "scope-n")
	require.True(t, ok)
	_, ok = registry.AnswerQuiz(offer.Token, "scope-n", 0)
	require.True(t, ok)

	registry.Clicked(dry(newcomer(clock)))
	registry.sweep()
	assert.Nil(t, quizOffered(t, events))
}

func TestAnOlderAccountWhoseBankRunsDryIsAskedNothing(t *testing.T) {
	registry, clock := newWelcomingRegistry(t)
	events := attend(t, registry, "scope-n")

	old := dry(newcomer(clock))
	old.Born = clock.Now().Add(-welcomeWithin)
	registry.Clicked(old)
	registry.sweep()

	assert.Nil(t, quizOffered(t, events))
}

func TestAnAccountThatSaysNothingOfItsAgeIsNotWelcomed(t *testing.T) {
	registry, clock := newWelcomingRegistry(t)
	events := attend(t, registry, "scope-n")

	unborn := dry(newcomer(clock))
	unborn.Born = time.Time{}
	registry.Clicked(unborn)
	registry.sweep()

	assert.Nil(t, quizOffered(t, events))
}

func TestWithNoWelcomeANewcomerWaitsLikeAnybody(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	events := attend(t, registry, "scope-n")

	registry.Clicked(dry(newcomer(clock)))
	registry.sweep()

	assert.Nil(t, quizOffered(t, events))
}

func TestANewcomerIsAskedNoWelcomeQuizWhileQuizzesAreOff(t *testing.T) {
	registry, clock := newWelcomingRegistry(t)
	events := attend(t, registry, "scope-n")
	rulesOf(registry).Enabled = false

	registry.Clicked(dry(newcomer(clock)))
	registry.sweep()
	rulesOf(registry).Enabled = true
	registry.sweep()

	assert.Nil(t, quizOffered(t, events))
}

func TestANewcomerIsForgottenOnceItIsNoLongerNew(t *testing.T) {
	registry, clock := newWelcomingRegistry(t)
	attend(t, registry, "scope-n")

	registry.Clicked(dry(newcomer(clock)))
	registry.sweep()
	require.Len(t, registry.newcomers, 1)

	clock.Advance(welcomeWithin)
	registry.sweep()
	assert.Empty(t, registry.newcomers)
}

func TestAFrozenSetLosesTheWelcome(t *testing.T) {
	registry, clock := newWelcomingRegistry(t)
	events := attend(t, registry, "scope-n")
	rulesOf(registry).Frozen = true

	registry.Clicked(dry(newcomer(clock)))
	registry.sweep()
	rulesOf(registry).Frozen = false
	registry.sweep()

	assert.Nil(t, quizOffered(t, events))
}
