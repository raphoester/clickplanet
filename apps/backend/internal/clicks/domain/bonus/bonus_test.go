package bonus

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestRegistry() (*Registry, *fakeClock) {
	clock := &fakeClock{now: epoch}
	return New(Config{Enabled: true, OfferTTL: 15 * time.Second, Duration: time.Minute, Multiplier: 3}, clock), clock
}

// offered drains the one event an attendee should have been sent.
func offered(t *testing.T, events <-chan Event) *Offer {
	t.Helper()

	select {
	case event := <-events:
		require.NotNil(t, event.Offer, "expected an offer, got %+v", event)
		return event.Offer
	default:
		return nil
	}
}

func TestNothingIsOfferedWithNobodyWatching(t *testing.T) {
	registry, _ := newTestRegistry()

	_, ok := registry.Offer()

	assert.False(t, ok, "a box offered to an empty room is only a token to sweep up later")
}

func TestTheOfferGoesToTheOneAttendeeDrawn(t *testing.T) {
	registry, _ := newTestRegistry()

	events, leave := registry.Attend("scope-a")
	defer leave()

	offer, ok := registry.Offer()
	require.True(t, ok)

	got := offered(t, events)
	require.NotNil(t, got)
	assert.Equal(t, offer.Token, got.Token)
}

func TestAnOfferReachesNobodyElse(t *testing.T) {
	registry, _ := newTestRegistry()

	// The whole point of addressing the offer: with it broadcast, the fastest
	// script always wins, and this would hand extra clicks to exactly the
	// callers the antibot package exists to stop.
	a, leaveA := registry.Attend("scope-a")
	defer leaveA()
	b, leaveB := registry.Attend("scope-b")
	defer leaveB()

	_, ok := registry.Offer()
	require.True(t, ok)

	drawnA := offered(t, a) != nil
	drawnB := offered(t, b) != nil

	assert.NotEqual(t, drawnA, drawnB, "exactly one of them should have been sent the box")
}

func TestManyTabsAreOneEntrantAndNotMany(t *testing.T) {
	registry, _ := newTestRegistry()

	_, leaveOne := registry.Attend("scope-a")
	_, leaveTwo := registry.Attend("scope-a")
	defer leaveOne()
	defer leaveTwo()

	assert.Equal(t, 1, registry.countAttendees())
}

func TestAnAttendeeStaysWhileAnyOfItsStreamsIsOpen(t *testing.T) {
	registry, _ := newTestRegistry()

	_, leaveOne := registry.Attend("scope-a")
	_, leaveTwo := registry.Attend("scope-a")

	leaveOne()
	assert.Equal(t, 1, registry.countAttendees(), "one tab closing is not the caller leaving")

	leaveTwo()
	assert.Equal(t, 0, registry.countAttendees())
}

func TestLeavingTwiceDoesNotStrandTheEntry(t *testing.T) {
	registry, _ := newTestRegistry()

	_, leave := registry.Attend("scope-a")
	leave()
	leave()

	assert.Equal(t, 0, registry.countAttendees())
}

func TestTheDrawReachesEveryAttendeeOverEnoughRounds(t *testing.T) {
	registry, _ := newTestRegistry()

	scopes := []string{"a", "b", "c", "d"}
	channels := map[string]<-chan Event{}
	for _, scope := range scopes {
		events, leave := registry.Attend(scope)
		defer leave()
		channels[scope] = events
	}

	// Uniform rather than weighted by how long a connection has been open:
	// rewarding an old connection rewards leaving a tab open, which is the
	// opposite of what catching a box is meant to reward.
	//
	// Every round is drained, or the per-attendee buffers saturate after eight
	// and this stops counting anything.
	const rounds = 2000

	seen := map[string]int{}
	for i := 0; i < rounds; i++ {
		_, ok := registry.Offer()
		require.True(t, ok)

		for scope, events := range channels {
			if offered(t, events) != nil {
				seen[scope]++
			}
		}
	}

	expected := rounds / len(scopes)
	for _, scope := range scopes {
		assert.InEpsilonf(t, expected, seen[scope], 0.25,
			"%s was drawn %d times out of %d, which is not an even share", scope, seen[scope], rounds)
	}
}

func TestAClaimByTheCallerItWasOfferedToSucceeds(t *testing.T) {
	registry, _ := newTestRegistry()

	_, leave := registry.Attend("scope-a")
	defer leave()

	offer, ok := registry.Offer()
	require.True(t, ok)

	reward, claimed := registry.Claim(offer.Token, "scope-a")

	require.True(t, claimed)
	assert.Equal(t, KindTripleClicks, reward.Kind)
	assert.Equal(t, time.Minute, reward.Duration)
}

func TestATokenIsWorthNothingToAnybodyElse(t *testing.T) {
	registry, _ := newTestRegistry()

	_, leave := registry.Attend("scope-a")
	defer leave()

	offer, ok := registry.Offer()
	require.True(t, ok)

	// Lifted off the wire, or guessed. Either way it is not this caller's.
	_, claimed := registry.Claim(offer.Token, "scope-b")
	assert.False(t, claimed)

	_, stillMine := registry.Claim(offer.Token, "scope-a")
	assert.True(t, stillMine, "a failed theft must not spend the real owner's box")
}

func TestATokenIsSpentExactlyOnce(t *testing.T) {
	registry, _ := newTestRegistry()

	_, leave := registry.Attend("scope-a")
	defer leave()

	offer, _ := registry.Offer()

	_, first := registry.Claim(offer.Token, "scope-a")
	_, second := registry.Claim(offer.Token, "scope-a")

	assert.True(t, first)
	assert.False(t, second)
}

func TestALapsedTokenIsRefused(t *testing.T) {
	registry, clock := newTestRegistry()

	_, leave := registry.Attend("scope-a")
	defer leave()

	offer, _ := registry.Offer()
	clock.advance(15 * time.Second)

	_, claimed := registry.Claim(offer.Token, "scope-a")

	assert.False(t, claimed)
}

func TestAnUnknownTokenIsRefused(t *testing.T) {
	registry, _ := newTestRegistry()

	_, claimed := registry.Claim("not-a-token", "scope-a")

	assert.False(t, claimed)
}

func TestLapsedTokensAreForgottenRatherThanKept(t *testing.T) {
	registry, clock := newTestRegistry()

	_, leave := registry.Attend("scope-a")
	defer leave()

	for i := 0; i < 5; i++ {
		registry.Offer()
	}
	require.Len(t, registry.offers, 5)

	clock.advance(time.Hour)
	registry.Offer()

	assert.Len(t, registry.offers, 1, "only the one just made should still be held")
}

func TestEveryTokenIsDifferent(t *testing.T) {
	registry, _ := newTestRegistry()

	_, leave := registry.Attend("scope-a")
	defer leave()

	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		offer, ok := registry.Offer()
		require.True(t, ok)
		require.False(t, seen[offer.Token], "a token was handed out twice")
		seen[offer.Token] = true
	}
}

func TestACatchIsAnnouncedToEveryone(t *testing.T) {
	registry, _ := newTestRegistry()

	a, leaveA := registry.Attend("scope-a")
	defer leaveA()
	b, leaveB := registry.Attend("scope-b")
	defer leaveB()

	registry.Publish(Taken{CountryID: "fr", Kind: KindTripleClicks})

	for _, events := range []<-chan Event{a, b} {
		select {
		case event := <-events:
			require.NotNil(t, event.Taken)
			assert.Equal(t, "fr", event.Taken.CountryID)
		default:
			t.Fatal("an attendee was not told about the catch")
		}
	}
}

func TestAnAttendeeThatIsNotReadingIsDroppedRatherThanBlocking(t *testing.T) {
	registry, _ := newTestRegistry()

	events, leave := registry.Attend("scope-a")
	defer leave()

	// Well past the buffer. A client that has stopped reading must cost the
	// draw nothing, exactly as a slow tile subscriber drops updates rather
	// than stalling the fanout.
	for i := 0; i < eventBuffer*4; i++ {
		registry.Offer()
	}

	assert.Equal(t, eventBuffer, len(events))
}

func TestTheDefaultsFillInWhatTheFileLeavesOut(t *testing.T) {
	registry := New(Config{Enabled: true}, &fakeClock{now: epoch})

	assert.Equal(t, defaultInterval, registry.config.Interval)
	assert.Equal(t, defaultDuration, registry.config.Duration)
	assert.Equal(t, float64(defaultMultiplier), registry.Multiplier())
}

func TestAMultiplierOfOneOrLessIsNotABonus(t *testing.T) {
	registry := New(Config{Enabled: true, Multiplier: 1}, &fakeClock{now: epoch})

	assert.Equal(t, float64(defaultMultiplier), registry.Multiplier())
}

// countAttendees reads what Offer draws from, under the same lock.
func (r *Registry) countAttendees() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.attendees)
}

func TestEveryTabOfOneCallerIsSentTheBox(t *testing.T) {
	registry, _ := newTestRegistry()

	// One entry in the draw, but the box appears in all of this caller's tabs.
	// A single shared channel would deliver it to whichever goroutine won the
	// receive, so the box would show up in a tab at random — possibly one the
	// player is not even looking at.
	first, leaveFirst := registry.Attend("scope-a")
	defer leaveFirst()
	second, leaveSecond := registry.Attend("scope-a")
	defer leaveSecond()

	require.Equal(t, 1, registry.countAttendees())

	offer, ok := registry.Offer()
	require.True(t, ok)

	assert.Equal(t, offer.Token, offered(t, first).Token)
	assert.Equal(t, offer.Token, offered(t, second).Token)
}

func TestOneTabClosingLeavesTheOthersReceiving(t *testing.T) {
	registry, _ := newTestRegistry()

	_, leaveFirst := registry.Attend("scope-a")
	second, leaveSecond := registry.Attend("scope-a")
	defer leaveSecond()

	leaveFirst()

	_, ok := registry.Offer()
	require.True(t, ok)

	assert.NotNil(t, offered(t, second), "the surviving tab should still be sent the box")
}
