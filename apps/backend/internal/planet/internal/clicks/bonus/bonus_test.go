package bonus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

const window = time.Minute

// A fixed window makes the schedule assertable; the spread has its own test.
func newTestRegistry() (*Registry, *cptime.FixedClock) {
	clock := cptime.NewFixedClock(epoch)

	return New(Config{
		Enabled:         true,
		MinInterval:     window,
		MaxInterval:     window,
		MissRetry:       20 * time.Second,
		OfferTTL:        15 * time.Second,
		Duration:        time.Minute,
		Multiplier:      3,
		ActiveWithin:    5 * time.Minute,
		ForgetAfter:     5 * time.Minute,
		MaxBoostPerHour: 15 * time.Minute,
		SweepInterval:   time.Second,
	}, clock), clock
}

// playing is a caller with a stream open who has clicked, which is what it
// takes to be offered anything.
func playing(t *testing.T, r *Registry, scope string) <-chan Event {
	t.Helper()

	events, leave := r.Attend(scope)
	t.Cleanup(leave)
	r.Clicked(scope)

	return events
}

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

func drain(events <-chan Event) {
	for {
		select {
		case <-events:
		default:
			return
		}
	}
}

// waitOut moves past a caller's whole window and sweeps, which is one turn.
func waitOut(r *Registry, clock *cptime.FixedClock) {
	clock.Advance(window + time.Second)
	r.sweep()
}

func TestABoxGoesToAnAttendingCallerOnceTheWindowPasses(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	require.Nil(t, offered(t, events), "nothing is due yet")

	waitOut(registry, clock)

	assert.NotNil(t, offered(t, events))
}

func TestEveryCallerIsOnTheirOwnScheduleRatherThanSharingOne(t *testing.T) {
	registry, clock := newTestRegistry()

	// The bug this replaces: one global ticker drew a single winner, so the
	// rate each player saw fell as 1/(interval × players).
	channels := map[string]<-chan Event{}
	for _, scope := range []string{"a", "b", "c", "d"} {
		channels[scope] = playing(t, registry, scope)
	}

	waitOut(registry, clock)

	for scope, events := range channels {
		assert.NotNilf(t, offered(t, events), "%s was not offered a box on its own turn", scope)
	}
}

func TestABoxReachesNobodyButTheCallerItWasDrawnFor(t *testing.T) {
	registry, clock := newTestRegistry()

	mine := playing(t, registry, "scope-a")
	theirs, leave := registry.Attend("scope-b")
	t.Cleanup(leave)

	// scope-b never clicked, so it is not playing and gets nothing.
	waitOut(registry, clock)

	assert.NotNil(t, offered(t, mine))
	assert.Nil(t, offered(t, theirs))
}

func TestEveryTabOfOneCallerIsSentTheBox(t *testing.T) {
	registry, clock := newTestRegistry()

	first := playing(t, registry, "scope-a")
	second, leave := registry.Attend("scope-a")
	t.Cleanup(leave)

	require.Len(t, registry.callers, 1, "tabs are one entrant, not many")

	waitOut(registry, clock)

	one, two := offered(t, first), offered(t, second)
	require.NotNil(t, one)
	require.NotNil(t, two)
	assert.Equal(t, one.Token, two.Token)
}

func TestNothingIsOfferedToACallerWhoIsNotClicking(t *testing.T) {
	registry, clock := newTestRegistry()

	events, leave := registry.Attend("scope-a")
	t.Cleanup(leave)

	waitOut(registry, clock)

	assert.Nil(t, offered(t, events), "a tab left open is not playing")
}

func TestATurnThatCameUpWhileAwayIsLostRatherThanBanked(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	// Past ActiveWithin, so several turns come and go unclaimed.
	clock.Advance(10 * time.Minute)
	registry.sweep()
	require.Nil(t, offered(t, events))

	// Clicking again does not hand over a backlog.
	registry.Clicked("scope-a")
	registry.sweep()
	assert.Nil(t, offered(t, events), "the slot was lost, not saved up")

	waitOut(registry, clock)
	assert.NotNil(t, offered(t, events))
}

func TestOnlyOneBoxIsOutstandingAtATime(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	require.NotNil(t, offered(t, events))

	clock.Advance(time.Second)
	registry.sweep()

	assert.Nil(t, offered(t, events))
}

func TestAMissedBoxBringsTheNextOneForwardOnce(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	require.NotNil(t, offered(t, events))

	// Let it lapse: the retry is due sooner than a fresh window would be.
	clock.Advance(16 * time.Second)
	registry.sweep()
	require.Nil(t, offered(t, events))

	clock.Advance(21 * time.Second)
	registry.sweep()

	assert.NotNil(t, offered(t, events), "a missed box should come back sooner")
}

func TestASecondMissInARowWaitsTheOrdinaryWindow(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	// Miss twice. Compounding the retry would hand a tab a box every
	// MissRetry for the rest of the session.
	for range 2 {
		waitOut(registry, clock)
		require.NotNil(t, offered(t, events))
		clock.Advance(16 * time.Second)
		registry.sweep()
	}

	clock.Advance(21 * time.Second)
	registry.sweep()
	assert.Nil(t, offered(t, events), "the second miss is not accelerated")

	waitOut(registry, clock)
	assert.NotNil(t, offered(t, events))
}

func TestCatchingOneHoldsTheNextUntilTheBonusIsOver(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, claimed := registry.Claim(offer.Token, "scope-a")
	require.True(t, claimed)

	// A window on its own would land a second box on a running bonus.
	clock.Advance(window + time.Second)
	registry.Clicked("scope-a")
	registry.sweep()
	assert.Nil(t, offered(t, events), "a bonus is still running")

	clock.Advance(window + time.Second)
	registry.Clicked("scope-a")
	registry.sweep()
	assert.NotNil(t, offered(t, events))
}

func TestCatchingOneClearsTheMissThatCameBefore(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	require.NotNil(t, offered(t, events))
	clock.Advance(16 * time.Second)
	registry.sweep()

	clock.Advance(21 * time.Second)
	registry.sweep()
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, claimed := registry.Claim(offer.Token, "scope-a")
	require.True(t, claimed)

	assert.Equal(t, 0, registry.callers["scope-a"].misses)
}

func TestReloadingCannotRerollTheSchedule(t *testing.T) {
	registry, clock := newTestRegistry()

	_, leave := registry.Attend("scope-a")
	registry.Clicked("scope-a")

	clock.Advance(30 * time.Second)
	due := registry.callers["scope-a"].nextOfferAt

	leave()
	events, second := registry.Attend("scope-a")
	t.Cleanup(second)

	assert.Equal(t, due, registry.callers["scope-a"].nextOfferAt)

	clock.Advance(29 * time.Second)
	registry.sweep()
	assert.Nil(t, offered(t, events), "the wait carried over the reconnect")
}

func TestAScheduleIsForgottenOnceTheCallerHasBeenGoneLongEnough(t *testing.T) {
	registry, clock := newTestRegistry()

	_, leave := registry.Attend("scope-a")
	leave()

	clock.Advance(4 * time.Minute)
	registry.sweep()
	require.Len(t, registry.callers, 1)

	clock.Advance(2 * time.Minute)
	registry.sweep()
	assert.Empty(t, registry.callers)
}

func TestTheHourlyCapStopsTheOffers(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	// Fifteen minutes of bonus at a minute each.
	for range 15 {
		clock.Advance(window + time.Second)
		registry.Clicked("scope-a")
		registry.sweep()

		offer := offered(t, events)
		require.NotNil(t, offer)

		_, claimed := registry.Claim(offer.Token, "scope-a")
		require.True(t, claimed)

		clock.Advance(time.Minute)
	}

	for range 5 {
		clock.Advance(window + time.Second)
		registry.Clicked("scope-a")
		registry.sweep()
		require.Nil(t, offered(t, events), "the cap should hold")
	}

	// It is an hour's cap, not a permanent one.
	clock.Advance(time.Hour)
	registry.Clicked("scope-a")
	registry.sweep()
	assert.NotNil(t, offered(t, events))
}

func TestTheWaitIsDrawnFromTheConfiguredWindow(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	registry := New(Config{
		Enabled: true, MinInterval: time.Minute, MaxInterval: 3 * time.Minute,
	}, clock)

	seen := map[time.Duration]bool{}
	for range 200 {
		drawn := registry.window()

		require.GreaterOrEqual(t, drawn, time.Minute)
		require.Less(t, drawn, 3*time.Minute)
		seen[drawn] = true
	}

	assert.Greater(t, len(seen), 100, "the wait should be spread, not fixed")
}

func TestAClaimByTheCallerItWasOfferedToSucceeds(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	reward, claimed := registry.Claim(offer.Token, "scope-a")

	require.True(t, claimed)
	assert.Equal(t, KindTripleClicks, reward.Kind)
	assert.Equal(t, time.Minute, reward.Duration)
}

func TestATokenIsWorthNothingToAnybodyElse(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, stolen := registry.Claim(offer.Token, "scope-b")
	assert.False(t, stolen)

	_, mine := registry.Claim(offer.Token, "scope-a")
	assert.True(t, mine, "a failed theft must not spend the owner's box")
}

func TestATokenIsSpentExactlyOnce(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, first := registry.Claim(offer.Token, "scope-a")
	_, second := registry.Claim(offer.Token, "scope-a")

	assert.True(t, first)
	assert.False(t, second)
}

func TestALapsedTokenIsRefused(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	clock.Advance(16 * time.Second)

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
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	require.NotNil(t, offered(t, events))

	clock.Advance(16 * time.Second)
	registry.sweep()

	assert.Empty(t, registry.offers)
}

func TestEveryTokenIsDifferent(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	seen := map[string]bool{}
	for range 20 {
		clock.Advance(window + time.Second)
		registry.Clicked("scope-a")
		registry.sweep()

		offer := offered(t, events)
		require.NotNil(t, offer)
		require.False(t, seen[offer.Token], "a token was handed out twice")
		seen[offer.Token] = true

		clock.Advance(16 * time.Second)
		registry.sweep()
	}
}

func TestACatchIsAnnouncedToEveryone(t *testing.T) {
	registry, _ := newTestRegistry()

	watchers := []<-chan Event{playing(t, registry, "scope-a"), playing(t, registry, "scope-b")}
	for _, events := range watchers {
		drain(events)
	}

	registry.Publish(Taken{CountryID: "fr", Kind: KindTripleClicks})

	for _, events := range watchers {
		select {
		case event := <-events:
			require.NotNil(t, event.Taken)
			assert.Equal(t, "fr", event.Taken.CountryID)
		default:
			t.Fatal("a caller was not told about the catch")
		}
	}
}

func TestACallerThatIsNotReadingIsDroppedRatherThanBlocking(t *testing.T) {
	registry, _ := newTestRegistry()
	events := playing(t, registry, "scope-a")

	for range eventBuffer * 4 {
		registry.Publish(Taken{CountryID: "fr", Kind: KindTripleClicks})
	}

	assert.Len(t, events, eventBuffer)
}

func TestTheDefaultsFillInWhatTheFileLeavesOut(t *testing.T) {
	registry := New(Config{Enabled: true}, cptime.NewFixedClock(epoch))

	assert.Equal(t, defaultMinInterval, registry.config.MinInterval)
	assert.Equal(t, defaultMaxInterval, registry.config.MaxInterval)
	assert.InDelta(t, float64(defaultMultiplier), registry.Multiplier(), 1e-9)
}

func TestAMaxBelowTheMinIsNotAWindow(t *testing.T) {
	registry := New(Config{
		Enabled: true, MinInterval: 10 * time.Minute, MaxInterval: time.Second,
	}, cptime.NewFixedClock(epoch))

	assert.GreaterOrEqual(t, registry.config.MaxInterval, registry.config.MinInterval)
	assert.Equal(t, 10*time.Minute, registry.window())
}
