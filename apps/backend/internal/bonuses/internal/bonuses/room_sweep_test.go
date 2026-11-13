package bonuses

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheRoomIsAskedWithTheRegistryUnlocked(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")
	// Clicked takes the registry's lock: asked under it, the sweep would never end.
	roomOf(registry).during = func() { clicked(registry, "scope-b") }

	clock.Advance(window + time.Second)
	swept := make(chan struct{})
	go func() {
		registry.sweep(t.Context())
		close(swept)
	}()

	select {
	case <-swept:
	case <-time.After(5 * time.Second):
		t.Fatal("the sweep held the registry while it asked the wallet")
	}
	assert.NotNil(t, offered(t, events))
}

func TestTheRoomIsAskedOfEveryAccountPlayingBehindACallerDue(t *testing.T) {
	registry, clock := newTestRegistry()
	attend(t, registry, "scope-a")
	registry.Clicked(Click{Caller: Caller{Scope: "scope-a", Account: "acc-1"}})
	registry.Clicked(Click{Caller: Caller{Scope: "scope-a", Account: "acc-2"}})
	registry.Clicked(Click{Caller: Caller{Scope: "scope-b", Account: "acc-3"}})

	waitOut(t, registry, clock)

	require.Len(t, roomOf(registry).asked, 1, "scope-b has no stream open, so it is not due and nothing is asked of it")
	assert.ElementsMatch(t, []Account{"acc-1", "acc-2"}, roomOf(registry).asked[0])
}

func TestARoomTheWalletCannotTellLosesTheSlot(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")
	roomOf(registry).Fail(errors.New("the database is down"))

	waitOut(t, registry, clock)
	require.Nil(t, offered(t, events), "a hand that cannot be read is offered nothing")
	assert.Equal(t, clock.Now().Add(window), registry.callers["scope-a"].nextOfferAt, "the slot is lost, as an away player's")

	roomOf(registry).Fail(nil)
	waitOut(t, registry, clock)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.NotNil(t, offered(t, events))
}

func TestAnAccountThatFirstClicksWhileTheRoomIsAskedWaitsForTheNextSweep(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	events := attend(t, registry, entrantOf(newcomerPayer))
	anonymous := Click{Caller: Caller{Scope: newcomerPayer.Scope}}
	registry.Clicked(anonymous)
	playing(t, registry, "scope-x")

	clock.Advance(ordinaryWindow + time.Second)
	registry.Clicked(anonymous)
	clicked(registry, "scope-x")
	// The newcomer's scope is due with no account behind it, and its first click lands while scope-x's room is asked.
	roomOf(registry).during = func() {
		roomOf(registry).during = nil
		registry.Clicked(clickOf(newcomerPayer))
	}
	registry.sweep(t.Context())
	require.Nil(t, offered(t, events), "the newcomer was not counted in a room asked")

	registry.sweep(t.Context())
	offer := offered(t, events)
	require.NotNil(t, offer, "its welcome box is not lost to the race")
	assert.Equal(t, welcomeBox, offer.Kind)
}
