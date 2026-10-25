package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAFrozenMapIsOfferedNoBoxUntilItThaws(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	registry.SetFrozen(true)
	waitOut(t, registry, clock)
	assert.Nil(t, offered(t, events))

	registry.SetFrozen(false)
	clock.Advance(time.Second)
	registry.sweep(t.Context())
	assert.NotNil(t, offered(t, events), "the box held back is due as soon as the map thaws")
}

func TestABoxOfferedBeforeTheFreezeLapses(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")
	waitOut(t, registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	registry.SetFrozen(true)
	registry.sweep(t.Context())

	_, claimed := registry.Claim(offer.Token, "scope-a", "scope-a")
	assert.False(t, claimed, "it was due to fly %s more", offer.ExpiresAt.Sub(clock.Now()))
}

func TestAFrozenMapIsAskedNoQuizAndTheQuizStandingLapses(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	_, _ = takeQuiz(t, registry, clock, "scope-a")
	events := attend(t, registry, "scope-b")
	clicked(registry, "scope-b")

	registry.SetFrozen(true)
	registry.sweep(t.Context())
	assert.Empty(t, registry.quizOffers, "the quiz opened before the freeze is gone")

	waitOutQuiz(t, registry, clock)
	assert.Nil(t, quizOffered(t, events))
}
