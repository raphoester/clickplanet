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
		SpreadDuration:  time.Minute,
		BombDuration:    time.Minute,
		EncloseDuration: time.Minute,
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

func TestEveryKindConfiguredIsOffered(t *testing.T) {
	registry, _ := newTestRegistry()

	seen := map[Kind]bool{}
	for range 200 {
		seen[registry.drawKind()] = true
	}

	assert.Len(t, seen, len(Kinds), "an empty bonus.kinds offers every kind")
}

func TestASpreadBoxRunsForItsOwnShorterDuration(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	registry := New(Config{
		Enabled: true, MinInterval: window, MaxInterval: window, ActiveWithin: 5 * time.Minute,
		Duration: time.Minute, SpreadDuration: 10 * time.Second,
		Kinds: map[Kind]float64{KindSpreadClicks: 1},
	}, clock)
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)
	assert.Equal(t, 10*time.Second, offer.Duration)

	reward, claimed := registry.Claim(offer.Token, "scope-a")
	require.True(t, claimed)
	assert.Equal(t, 10*time.Second, reward.Duration)
}

func bombRegistry() (*Registry, *cptime.FixedClock) {
	clock := cptime.NewFixedClock(epoch)

	return New(Config{
		Enabled: true, MinInterval: window, MaxInterval: window, ActiveWithin: 5 * time.Minute,
		BombDuration: 30 * time.Second,
		Kinds:        map[Kind]float64{KindBomb: 1},
	}, clock), clock
}

func TestABombBoxIsHeldForTheBombsOwnDuration(t *testing.T) {
	registry, clock := bombRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	reward, claimed := registry.Claim(offer.Token, "scope-a")
	require.True(t, claimed)
	assert.Equal(t, KindBomb, reward.Kind)
	assert.Equal(t, 30*time.Second, reward.Duration)
}

func TestDroppingABombBringsTheNextBoxToAWindowFromTheDrop(t *testing.T) {
	registry, clock := bombRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)
	_, claimed := registry.Claim(offer.Token, "scope-a")
	require.True(t, claimed)

	clock.Advance(5 * time.Second)
	registry.Dropped("scope-a")

	assert.Equal(t, clock.Now().Add(window), registry.callers["scope-a"].nextOfferAt,
		"not a window after the 25 seconds the bomb could still have been held")
}

func TestTheHourlyCapCountsTheTimeEachBonusActuallyRan(t *testing.T) {
	registry, clock := newTestRegistry()
	entry := registry.caller("scope-a", clock.Now())

	// Five ten-second spreads are under a minute, far from a fifteen-minute cap.
	for range 5 {
		entry.grants = append(entry.grants, grant{at: clock.Now(), duration: 10 * time.Second})
	}

	assert.False(t, registry.capped(entry, clock.Now()))
}

func TestAKindLeftOutOrAtZeroIsNeverOffered(t *testing.T) {
	for _, kinds := range []map[Kind]float64{
		{KindSpreadClicks: 1},
		{KindSpreadClicks: 1, KindTripleClicks: 0},
	} {
		registry := New(Config{Enabled: true, Kinds: kinds}, cptime.NewFixedClock(epoch))

		for range 50 {
			require.Equal(t, KindSpreadClicks, registry.drawKind())
		}
	}
}

func TestKindsAreDrawnInProportionToTheirWeight(t *testing.T) {
	registry := New(Config{
		Enabled: true,
		Kinds:   map[Kind]float64{KindTripleClicks: 9, KindSpreadClicks: 1},
	}, cptime.NewFixedClock(epoch))

	const draws = 20_000
	spreads := 0
	for range draws {
		if registry.drawKind() == KindSpreadClicks {
			spreads++
		}
	}

	// One in ten, give or take far more than the noise of 20,000 draws.
	assert.InDelta(t, 0.1, float64(spreads)/draws, 0.02)
}

func TestKindWeightsThatMakeNoSenseRefuseTheConfig(t *testing.T) {
	require.NoError(t, Config{Kinds: map[Kind]float64{KindTripleClicks: 4, KindSpreadClicks: 1}}.Validate())
	require.NoError(t, Config{Kinds: map[Kind]float64{KindTripleClicks: 1, KindSpreadClicks: 0}}.Validate())
	require.NoError(t, Config{}.Validate())

	assert.ErrorContains(t, Config{Kinds: map[Kind]float64{"quadruple_clicks": 1}}.Validate(), "quadruple_clicks")
	assert.ErrorContains(t, Config{Kinds: map[Kind]float64{KindSpreadClicks: -1}}.Validate(), "spread_clicks")
	assert.ErrorContains(t, Config{Kinds: map[Kind]float64{KindTripleClicks: 0}}.Validate(), "weight of 0")
}

func TestAClaimByTheCallerItWasOfferedToSucceeds(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	reward, claimed := registry.Claim(offer.Token, "scope-a")

	require.True(t, claimed)
	assert.Equal(t, offer.Kind, reward.Kind, "the claim grants what the box said it was")
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

func TestAnEncloseBoxRunsForItsOwnDurationAndSaysHowManyShapes(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	registry := New(Config{
		Enabled: true, MinInterval: window, MaxInterval: window, ActiveWithin: 5 * time.Minute,
		Duration: time.Minute, EncloseDuration: 30 * time.Second, EncloseShapes: 3, EncloseMaxTiles: 10,
		Kinds: map[Kind]float64{KindEncloseClicks: 1},
	}, clock)
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)
	assert.Equal(t, 30*time.Second, offer.Duration)

	reward, claimed := registry.Claim(offer.Token, "scope-a")
	require.True(t, claimed)
	assert.Equal(t, Reward{
		Kind: KindEncloseClicks, Duration: 30 * time.Second, Enclosures: 3, EnclosureMaxTiles: 10,
	}, reward)
}

func TestOnlyAnEncloseRewardCarriesShapes(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	registry := New(Config{
		Enabled: true, MinInterval: window, MaxInterval: window, ActiveWithin: 5 * time.Minute,
		Kinds: map[Kind]float64{KindTripleClicks: 1},
	}, clock)
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	reward, claimed := registry.Claim(offer.Token, "scope-a")
	require.True(t, claimed)
	assert.Zero(t, reward.Enclosures)
	assert.Zero(t, reward.EnclosureMaxTiles)
}

func TestAClosedShapeReachesEveryoneAndOnlyItsCloserIsToldItIsTheirs(t *testing.T) {
	registry, _ := newTestRegistry()
	mine, leaveMine := registry.Attend("scope-a")
	t.Cleanup(leaveMine)
	theirs, leaveTheirs := registry.Attend("scope-b")
	t.Cleanup(leaveTheirs)

	registry.PublishEnclosed("scope-a", Enclosed{
		CountryID: "fr", ClosingTile: 7, Wall: []uint32{7, 8}, Filled: []uint32{9}, Left: 2,
	})

	yours := (<-mine).Enclosed
	require.NotNil(t, yours)
	assert.True(t, yours.Yours)
	assert.Equal(t, 2, yours.Left)
	assert.Equal(t, []uint32{9}, yours.Filled)

	seen := (<-theirs).Enclosed
	require.NotNil(t, seen)
	assert.False(t, seen.Yours)
	assert.Zero(t, seen.Left, "how many shapes somebody has left is theirs to know")
	assert.Equal(t, "fr", seen.CountryID)
	assert.Equal(t, []uint32{7, 8}, seen.Wall)
}
