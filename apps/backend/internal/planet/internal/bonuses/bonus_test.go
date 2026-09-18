package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

const window = time.Minute

// A fixed window makes the schedule assertable; the spread has its own test. Only refills are offered,
// and the holdings are the test's own, so a claim never holds a kind back.
func newTestRegistry() (*Registry, *cptime.FixedClock) {
	return newRegistryOffering(map[Kind]float64{KindRefill: 1})
}

func newRegistryOffering(kinds map[Kind]float64) (*Registry, *cptime.FixedClock) {
	clock := cptime.NewFixedClock(epoch)

	return New(Config{
		MinInterval:       window,
		MaxInterval:       window,
		MissRetry:         20 * time.Second,
		OfferTTL:          15 * time.Second,
		Kinds:             kinds,
		ActiveWithin:      5 * time.Minute,
		ForgetAfter:       5 * time.Minute,
		MaxChargesPerHour: 6,
		ChargeTTL:         24 * time.Hour,
		SweepInterval:     time.Second,
	}, clock, newFakeHoldings()), clock
}

// holderOf is the account that plays from scope: one each, here.
func holderOf(scope string) Holder {
	return HolderOf(clicks.Payer{Scope: scope, Account: "acc-" + scope})
}

// fakeHoldings is what each holder holds, set by the test.
type fakeHoldings struct{ held map[Holder]Held }

func newFakeHoldings() *fakeHoldings {
	return &fakeHoldings{held: map[Holder]Held{}}
}

func (f *fakeHoldings) Held(holder Holder) Held { return f.held[holder] }

func (f *fakeHoldings) grant(holder Holder, kind Kind) {
	held := f.held[holder]
	switch kind {
	case KindRefill:
		held.Refill = true
	case KindBomb:
		held.Bomb = true
	case KindEncloseClicks:
		held.Enclose = true
	case KindSpreadClicks:
		held.SpreadClicks = 8
	}
	f.held[holder] = held
}

// take spends everything holder holds.
func (f *fakeHoldings) take(holder Holder) {
	delete(f.held, holder)
}

func holdingsOf(r *Registry) *fakeHoldings {
	return r.holdings.(*fakeHoldings) //nolint:forcetypeassert // every registry in these tests is built with one.
}

// attend opens a stream, closed when the test ends.
func attend(t *testing.T, r *Registry, scope string) <-chan Event {
	t.Helper()

	events, leave := r.Attend(scope)
	t.Cleanup(leave)

	return events
}

// clicked is a click from scope by a caller with no account.
func clicked(r *Registry, scope string) {
	r.Clicked(scope, holderOf(scope))
}

// playing is a caller with a stream open who has clicked, which is what it
// takes to be offered anything.
func playing(t *testing.T, r *Registry, scope string) <-chan Event {
	t.Helper()

	events := attend(t, r, scope)
	clicked(r, scope)

	return events
}

// everyKind is every kind, allowed.
func everyKind() *cpcolls.Set[Kind] {
	return cpcolls.NewSet(Kinds...)
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
	theirs := attend(t, registry, "scope-b")

	// scope-b never clicked, so it is not playing and gets nothing.
	waitOut(registry, clock)

	assert.NotNil(t, offered(t, mine))
	assert.Nil(t, offered(t, theirs))
}

func TestEveryTabOfOneCallerIsSentTheBox(t *testing.T) {
	registry, clock := newTestRegistry()

	first := playing(t, registry, "scope-a")
	second := attend(t, registry, "scope-a")

	require.Len(t, registry.callers, 1, "tabs are one entrant, not many")

	waitOut(registry, clock)

	one, two := offered(t, first), offered(t, second)
	require.NotNil(t, one)
	require.NotNil(t, two)
	assert.Equal(t, one.Token, two.Token)
}

func TestNothingIsOfferedToACallerWhoIsNotClicking(t *testing.T) {
	registry, clock := newTestRegistry()

	events := attend(t, registry, "scope-a")

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
	clicked(registry, "scope-a")
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
	clicked(registry, "scope-a")

	clock.Advance(30 * time.Second)
	due := registry.callers["scope-a"].nextOfferAt

	leave()
	events := attend(t, registry, "scope-a")

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

	// A script that catches every box. Nothing is ever held here, so only the cap stops it.
	for range 6 {
		clock.Advance(window + time.Second)
		clicked(registry, "scope-a")
		registry.sweep()

		offer := offered(t, events)
		require.NotNil(t, offer)

		_, claimed := registry.Claim(offer.Token, "scope-a")
		require.True(t, claimed)
	}

	for range 5 {
		clock.Advance(window + time.Second)
		clicked(registry, "scope-a")
		registry.sweep()
		require.Nil(t, offered(t, events), "the cap should hold")
	}

	// It is an hour's cap, not a permanent one.
	clock.Advance(time.Hour)
	clicked(registry, "scope-a")
	registry.sweep()
	assert.NotNil(t, offered(t, events))
}

func TestTheWaitIsDrawnFromTheConfiguredWindow(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	registry := New(Config{
		MinInterval: time.Minute, MaxInterval: 3 * time.Minute,
	}, clock, newFakeHoldings())

	seen := cpcolls.NewSet[time.Duration]()
	for range 200 {
		drawn := registry.window()

		require.GreaterOrEqual(t, drawn, time.Minute)
		require.Less(t, drawn, 3*time.Minute)
		seen.Add(drawn)
	}

	assert.Greater(t, seen.Len(), 100, "the wait should be spread, not fixed")
}

func TestEveryKindConfiguredIsOffered(t *testing.T) {
	registry := New(Config{}, cptime.NewFixedClock(epoch), newFakeHoldings())

	seen := cpcolls.NewSet[Kind]()
	for range 200 {
		seen.Add(registry.drawKind(everyKind()))
	}

	assert.Equal(t, len(Kinds), seen.Len(), "an empty bonus.kinds takes the defaults, which offer every kind")
}

func TestAnEmptyKindsTakesTheDefaultWeights(t *testing.T) {
	registry := New(Config{}, cptime.NewFixedClock(epoch), newFakeHoldings())

	assert.Equal(t, map[Kind]float64{
		KindRefill:        5,
		KindSpreadClicks:  3,
		KindEncloseClicks: 2,
		KindBomb:          1,
	}, registry.config.Kinds)
}

func TestEveryKindIsClaimedAsItself(t *testing.T) {
	for _, kind := range Kinds {
		registry, clock := newRegistryOffering(map[Kind]float64{kind: 1})
		events := playing(t, registry, "scope-a")

		waitOut(registry, clock)
		offer := offered(t, events)
		require.NotNil(t, offer)
		assert.Equal(t, kind, offer.Kind)

		reward, claimed := registry.Claim(offer.Token, "scope-a")
		require.True(t, claimed)
		assert.Equal(t, Reward{Kind: kind}, reward)
	}
}

func TestCatchingAChargeBringsTheNextBoxAWindowAfterTheClaim(t *testing.T) {
	registry, clock := newRegistryOffering(map[Kind]float64{KindBomb: 1})
	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, claimed := registry.Claim(offer.Token, "scope-a")
	require.True(t, claimed)

	assert.Equal(t, clock.Now().Add(window), registry.callers["scope-a"].nextOfferAt,
		"a charge has no end to wait for, so holding a bomb does not hold back every other box")
}

func TestAKindHeldIsNotOfferedAgainUntilItIsSpent(t *testing.T) {
	registry, clock := newRegistryOffering(map[Kind]float64{KindBomb: 1})
	events := playing(t, registry, "scope-a")
	holdingsOf(registry).grant(holderOf("scope-a"), KindBomb)

	for range 5 {
		clock.Advance(window + time.Second)
		clicked(registry, "scope-a")
		registry.sweep()
		require.Nil(t, offered(t, events), "a second bomb would be a stockpile")
	}

	holdingsOf(registry).take(holderOf("scope-a"))

	waitOut(registry, clock)
	clicked(registry, "scope-a")
	registry.sweep()
	assert.NotNil(t, offered(t, events), "a bomb dropped is a bomb that may be offered again")
}

func TestAKindHeldIsLeftOutOfTheDrawAndTheOthersStillCome(t *testing.T) {
	registry, clock := newRegistryOffering(map[Kind]float64{KindBomb: 100, KindRefill: 1})
	events := playing(t, registry, "scope-a")
	holdingsOf(registry).grant(holderOf("scope-a"), KindBomb)

	for range 20 {
		clock.Advance(window + time.Second)
		clicked(registry, "scope-a")
		registry.sweep()

		offer := offered(t, events)
		require.NotNil(t, offer)
		require.Equal(t, KindRefill, offer.Kind)

		clock.Advance(16 * time.Second)
		registry.sweep()
	}
}

func TestAKindHeldByAnAccountThatClicksFromTheScopeIsNotOffered(t *testing.T) {
	registry, clock := newRegistryOffering(map[Kind]float64{KindEncloseClicks: 1})
	account := HolderOf(clicks.Payer{Scope: "scope-a", Account: "acc-1"})

	events := attend(t, registry, "scope-a")
	registry.Clicked("scope-a", account)
	holdingsOf(registry).grant(account, KindEncloseClicks)

	waitOut(registry, clock)
	assert.Nil(t, offered(t, events), "the charge is the account's, whatever address it plays from")
}

func TestAPlayerWhoStoppedClickingNoLongerHoldsBackAKind(t *testing.T) {
	registry, clock := newRegistryOffering(map[Kind]float64{KindBomb: 1})
	gone := HolderOf(clicks.Payer{Scope: "scope-a", Account: "gone"})

	events := attend(t, registry, "scope-a")
	registry.Clicked("scope-a", gone)
	holdingsOf(registry).grant(gone, KindBomb)

	// Six minutes on, only somebody else behind the address is playing.
	clock.Advance(6 * time.Minute)
	clicked(registry, "scope-a")

	waitOut(registry, clock)
	clicked(registry, "scope-a")
	registry.sweep()
	assert.NotNil(t, offered(t, events), "a bomb held by somebody who left is not this player's")
}

func TestACallerWithNoAccountIsOfferedNothing(t *testing.T) {
	registry, clock := newTestRegistry()
	events := attend(t, registry, "scope-a")

	for range 5 {
		clock.Advance(window + time.Second)
		registry.Clicked("scope-a", NoHolder)
		registry.sweep()

		require.Nil(t, offered(t, events), "every bonus is a charge, and only an account can hold one")
	}
}

func TestTheHourlyCapCountsOnlyTheLastHour(t *testing.T) {
	registry, clock := newTestRegistry()
	entry := registry.caller("scope-a", clock.Now())

	entry.grants = append(entry.grants, clock.Now().Add(-2*time.Hour), clock.Now().Add(-time.Minute), clock.Now())

	assert.Equal(t, 2, registry.grantedWithinTheHour(entry, clock.Now()))
	assert.Len(t, entry.grants, 2, "an older grant is forgotten on the way")
}

func TestAKindLeftOutOrAtZeroIsNeverOffered(t *testing.T) {
	for _, kinds := range []map[Kind]float64{
		{KindSpreadClicks: 1},
		{KindSpreadClicks: 1, KindRefill: 0},
	} {
		registry, clock := newRegistryOffering(kinds)
		clicked(registry, "scope-a")
		entry := registry.callers["scope-a"]

		assert.Equal(t, cpcolls.NewSet(KindSpreadClicks), registry.offerable(entry, clock.Now()))
	}
}

func TestKindsAreDrawnInProportionToTheirWeight(t *testing.T) {
	registry := New(Config{
		Kinds: map[Kind]float64{KindRefill: 9, KindSpreadClicks: 1},
	}, cptime.NewFixedClock(epoch), newFakeHoldings())

	const draws = 20_000
	spreads := 0
	for range draws {
		if registry.drawKind(everyKind()) == KindSpreadClicks {
			spreads++
		}
	}

	// One in ten, give or take far more than the noise of 20,000 draws.
	assert.InDelta(t, 0.1, float64(spreads)/draws, 0.02)
}

func TestKindWeightsThatMakeNoSenseRefuseTheConfig(t *testing.T) {
	require.NoError(t, Config{Kinds: map[Kind]float64{KindRefill: 4, KindSpreadClicks: 1}}.Validate())
	require.NoError(t, Config{Kinds: map[Kind]float64{KindRefill: 1, KindSpreadClicks: 0}}.Validate())
	require.NoError(t, Config{}.Validate())

	assert.ErrorContains(t, Config{Kinds: map[Kind]float64{"quadruple_clicks": 1}}.Validate(), "quadruple_clicks")
	assert.ErrorContains(t, Config{Kinds: map[Kind]float64{KindSpreadClicks: -1}}.Validate(), "spread_clicks")
	assert.ErrorContains(t, Config{Kinds: map[Kind]float64{KindRefill: 0}}.Validate(), "weight of 0")
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
}

func TestACatchIsReportedWithHowLongItTook(t *testing.T) {
	registry, clock := newTestRegistry()

	var (
		caughtBy string
		after    time.Duration
	)
	registry.Observe(Report{Caught: func(scope string, took time.Duration) { caughtBy, after = scope, took }})

	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	clock.Advance(1200 * time.Millisecond)
	_, claimed := registry.Claim(offer.Token, "scope-a")
	require.True(t, claimed)

	assert.Equal(t, "scope-a", caughtBy)
	assert.Equal(t, 1200*time.Millisecond, after, "from the offer being sent, not from the sweep's window")
}

func TestARefusedClaimIsNotACatch(t *testing.T) {
	registry, clock := newTestRegistry()

	caught := 0
	registry.Observe(Report{Caught: func(string, time.Duration) { caught++ }})

	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, stolen := registry.Claim(offer.Token, "scope-b")
	require.False(t, stolen)

	assert.Zero(t, caught)
}

func TestALapsedBoxIsReportedAgainstItsCaller(t *testing.T) {
	registry, clock := newTestRegistry()

	var missedBy []string
	registry.Observe(Report{Lapsed: func(scope string) { missedBy = append(missedBy, scope) }})

	events := playing(t, registry, "scope-a")

	waitOut(registry, clock)
	require.NotNil(t, offered(t, events))

	clock.Advance(16 * time.Second)
	registry.sweep()

	assert.Equal(t, []string{"scope-a"}, missedBy)
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

	seen := cpcolls.NewSet[string]()
	for range 20 {
		clock.Advance(window + time.Second)
		clicked(registry, "scope-a")
		registry.sweep()

		offer := offered(t, events)
		require.NotNil(t, offer)
		require.False(t, seen.Contains(offer.Token), "a token was handed out twice")
		seen.Add(offer.Token)

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

	registry.Publish(Taken{CountryID: "fr", Kind: KindRefill})

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

func TestASpreadClickIsAnnouncedToEveryone(t *testing.T) {
	registry, _ := newTestRegistry()

	watchers := []<-chan Event{playing(t, registry, "scope-a"), playing(t, registry, "scope-b")}
	for _, events := range watchers {
		drain(events)
	}

	registry.PublishSpread(Spread{CountryID: "fr", Tile: 100, Neighbours: []uint32{99, 101}})

	for _, events := range watchers {
		require.Len(t, events, 1)

		spread := <-events
		require.NotNil(t, spread.Spread)
		assert.Equal(t, Spread{CountryID: "fr", Tile: 100, Neighbours: []uint32{99, 101}}, *spread.Spread)
	}
}

func TestACallerThatIsNotReadingIsDroppedRatherThanBlocking(t *testing.T) {
	registry, _ := newTestRegistry()
	events := playing(t, registry, "scope-a")

	for range eventBuffer * 4 {
		registry.Publish(Taken{CountryID: "fr", Kind: KindRefill})
	}

	assert.Len(t, events, eventBuffer)
}

func TestTheDefaultsFillInWhatTheFileLeavesOut(t *testing.T) {
	registry := New(Config{}, cptime.NewFixedClock(epoch), newFakeHoldings())

	assert.Equal(t, defaultMinInterval, registry.config.MinInterval)
	assert.Equal(t, defaultMaxInterval, registry.config.MaxInterval)
	assert.Equal(t, defaultMaxChargesPerHour, registry.config.MaxChargesPerHour)
}

func TestAMaxBelowTheMinIsNotAWindow(t *testing.T) {
	registry := New(Config{
		MinInterval: 10 * time.Minute, MaxInterval: time.Second,
	}, cptime.NewFixedClock(epoch), newFakeHoldings())

	assert.GreaterOrEqual(t, registry.config.MaxInterval, registry.config.MinInterval)
	assert.Equal(t, 10*time.Minute, registry.window())
}

func TestAClosedShapeReachesEveryoneAndOnlyItsCloserIsToldItIsTheirs(t *testing.T) {
	registry, _ := newTestRegistry()
	mine := attend(t, registry, "scope-a")
	theirs := attend(t, registry, "scope-b")

	registry.PublishEnclosed("scope-a", Enclosed{
		CountryID: "fr", ClosingTile: 7, Wall: []uint32{7, 8}, Filled: []uint32{9},
	})

	yours := (<-mine).Enclosed
	require.NotNil(t, yours)
	assert.True(t, yours.Yours)
	assert.Equal(t, []uint32{9}, yours.Filled)

	seen := (<-theirs).Enclosed
	require.NotNil(t, seen)
	assert.False(t, seen.Yours)
	assert.Equal(t, "fr", seen.CountryID)
	assert.Equal(t, []uint32{7, 8}, seen.Wall)
}
