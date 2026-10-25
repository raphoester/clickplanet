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

	registry.SetPace(Pace{MinInterval: 10 * time.Second, MaxInterval: 10 * time.Second, MaxChargesPerHour: 6})
	clock.Advance(11 * time.Second)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())

	assert.NotNil(t, offered(t, events), "due a minute after the first click, now ten seconds after the pace changed")
}

func TestASlowerPaceKeepsTheBoxAlreadyDueAndSpacesTheNext(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	registry.SetPace(Pace{MinInterval: 5 * time.Minute, MaxInterval: 5 * time.Minute, MaxChargesPerHour: 6})
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
	registry.SetPace(Pace{MinInterval: window, MaxInterval: window, MaxChargesPerHour: 8})

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

func TestNoBoxIsOfferedBeforeAPaceIsSet(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	registry := New(Config{Kinds: map[Kind]float64{KindRefill: 1}, ActiveWithin: 5 * time.Minute}, clock, NewMemoryStore())
	events := playing(t, registry, "scope-a")

	for range 3 {
		clock.Advance(time.Minute)
		clicked(registry, "scope-a")
		registry.sweep(t.Context())
	}

	assert.Nil(t, offered(t, events), "a cap of 0 an hour offers nothing")
}
