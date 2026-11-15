package boxes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

const window = time.Minute

func newTestRegistry() (*Registry, *cptime.FixedClock) {
	return newRuledRegistry(func(*Rules) {})
}

func newRuledRegistry(change func(*Rules)) (*Registry, *cptime.FixedClock) {
	clock := cptime.NewFixedClock(epoch)

	rules := TestRules()
	rules.Pace = Pace{MinInterval: window, MaxInterval: window, MaxBoxesPerHour: 6}
	rules.MissRetry = 20 * time.Second
	rules.OfferTTL = 15 * time.Second
	rules.ActiveWithin = 5 * time.Minute
	change(&rules)

	read := &rules
	registry := New(Config{ForgetAfter: 5 * time.Minute, SweepInterval: time.Second}, clock, &fakeRoom{},
		func() Rules { return *read })
	testRules[registry] = read

	return registry, clock
}

// The rules each registry of these tests reads, for a test to change.
var testRules = map[*Registry]*Rules{}

func rulesOf(r *Registry) *Rules {
	return testRules[r]
}

func accountOf(entrant Entrant) Account {
	return Account("acc-" + string(entrant))
}

func roomOf(r *Registry) *fakeRoom {
	return r.room.(*fakeRoom) //nolint:forcetypeassert // every registry in these tests is built with one.
}

func attend(t *testing.T, r *Registry, entrant Entrant) <-chan Offer {
	t.Helper()

	events, leave := r.Attend(entrant)
	t.Cleanup(leave)

	return events
}

func clicked(r *Registry, entrant Entrant) {
	r.Clicked(Click{Caller: Caller{Scope: string(entrant), Account: accountOf(entrant)}})
}

func playing(t *testing.T, r *Registry, entrant Entrant) <-chan Offer {
	t.Helper()

	events := attend(t, r, entrant)
	clicked(r, entrant)

	return events
}

func offered(t *testing.T, offers <-chan Offer) *Offer {
	t.Helper()

	select {
	case offer := <-offers:
		return &offer
	default:
		return nil
	}
}

func waitOut(t *testing.T, r *Registry, clock *cptime.FixedClock) {
	t.Helper()

	clock.Advance(window + time.Second)
	r.sweep(t.Context())
}

func TestABoxGoesToAnAttendingCallerOnceTheWindowPasses(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	require.Nil(t, offered(t, events), "nothing is due yet")

	waitOut(t, registry, clock)

	assert.NotNil(t, offered(t, events))
}

func TestEveryCallerIsOnTheirOwnScheduleRatherThanSharingOne(t *testing.T) {
	registry, clock := newTestRegistry()

	channels := map[Entrant]<-chan Offer{}
	for _, entrant := range []Entrant{"a", "b", "c", "d"} {
		channels[entrant] = playing(t, registry, entrant)
	}

	waitOut(t, registry, clock)

	for entrant, events := range channels {
		assert.NotNilf(t, offered(t, events), "%s was not offered a box on its own turn", entrant)
	}
}

func TestABoxReachesNobodyButTheCallerItWasDrawnFor(t *testing.T) {
	registry, clock := newTestRegistry()

	mine := playing(t, registry, "scope-a")
	theirs := attend(t, registry, "scope-b")

	waitOut(t, registry, clock)

	assert.NotNil(t, offered(t, mine))
	assert.Nil(t, offered(t, theirs))
}

func TestEveryTabOfOneCallerIsSentTheBox(t *testing.T) {
	registry, clock := newTestRegistry()

	first := playing(t, registry, "scope-a")
	second := attend(t, registry, "scope-a")

	require.Len(t, registry.callers, 1, "tabs are one entrant, not many")

	waitOut(t, registry, clock)

	one, two := offered(t, first), offered(t, second)
	require.NotNil(t, one)
	require.NotNil(t, two)
	assert.Equal(t, one.Token, two.Token)
}

func TestNothingIsOfferedToACallerWhoIsNotClicking(t *testing.T) {
	registry, clock := newTestRegistry()

	events := attend(t, registry, "scope-a")

	waitOut(t, registry, clock)

	assert.Nil(t, offered(t, events), "a tab left open is not playing")
}

func TestATurnThatCameUpWhileAwayIsLostRatherThanBanked(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	clock.Advance(10 * time.Minute)
	registry.sweep(t.Context())
	require.Nil(t, offered(t, events))

	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.Nil(t, offered(t, events), "the slot was lost, not saved up")

	waitOut(t, registry, clock)
	assert.NotNil(t, offered(t, events))
}

func TestOnlyOneBoxIsOutstandingAtATime(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	require.NotNil(t, offered(t, events))

	clock.Advance(time.Second)
	registry.sweep(t.Context())

	assert.Nil(t, offered(t, events))
}

func TestAMissedBoxBringsTheNextOneForwardOnce(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	require.NotNil(t, offered(t, events))

	clock.Advance(16 * time.Second)
	registry.sweep(t.Context())
	require.Nil(t, offered(t, events))

	clock.Advance(21 * time.Second)
	registry.sweep(t.Context())

	assert.NotNil(t, offered(t, events), "a missed box should come back sooner")
}

func TestASecondMissInARowWaitsTheOrdinaryWindow(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	for range 2 {
		waitOut(t, registry, clock)
		require.NotNil(t, offered(t, events))
		clock.Advance(16 * time.Second)
		registry.sweep(t.Context())
	}

	clock.Advance(21 * time.Second)
	registry.sweep(t.Context())
	assert.Nil(t, offered(t, events), "the second miss is not accelerated")

	waitOut(t, registry, clock)
	assert.NotNil(t, offered(t, events))
}

func TestCatchingOneClearsTheMissThatCameBefore(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	require.NotNil(t, offered(t, events))
	clock.Advance(16 * time.Second)
	registry.sweep(t.Context())

	clock.Advance(21 * time.Second)
	registry.sweep(t.Context())
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, claimed := registry.Claim(offer.Token, "scope-a", "scope-a")
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
	registry.sweep(t.Context())
	assert.Nil(t, offered(t, events), "the wait carried over the reconnect")
}

func TestAScheduleIsForgottenOnceTheCallerHasBeenGoneLongEnough(t *testing.T) {
	registry, clock := newTestRegistry()

	_, leave := registry.Attend("scope-a")
	leave()

	clock.Advance(4 * time.Minute)
	registry.sweep(t.Context())
	require.Len(t, registry.callers, 1)

	clock.Advance(2 * time.Minute)
	registry.sweep(t.Context())
	assert.Empty(t, registry.callers)
}

func TestTheHourlyCapStopsTheOffers(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	for range 6 {
		clock.Advance(window + time.Second)
		clicked(registry, "scope-a")
		registry.sweep(t.Context())

		offer := offered(t, events)
		require.NotNil(t, offer)

		_, claimed := registry.Claim(offer.Token, "scope-a", "scope-a")
		require.True(t, claimed)
	}

	for range 5 {
		clock.Advance(window + time.Second)
		clicked(registry, "scope-a")
		registry.sweep(t.Context())
		require.Nil(t, offered(t, events), "the cap should hold")
	}

	clock.Advance(time.Hour)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.NotNil(t, offered(t, events))
}

func TestTheWaitIsDrawnFromThePacesWindow(t *testing.T) {
	rules := TestRules()
	rules.Pace = Pace{MinInterval: time.Minute, MaxInterval: 3 * time.Minute, MaxBoxesPerHour: 12}

	seen := cpcolls.NewSet[time.Duration]()
	for range 200 {
		drawn := boxWait(rules)

		require.GreaterOrEqual(t, drawn, time.Minute)
		require.Less(t, drawn, 3*time.Minute)
		seen.Add(drawn)
	}

	assert.Greater(t, seen.Len(), 100, "the wait should be spread, not fixed")
}

func TestCatchingABoxBringsTheNextBoxAWindowAfterTheClaim(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, claimed := registry.Claim(offer.Token, "scope-a", "scope-a")
	require.True(t, claimed)

	assert.Equal(t, clock.Now().Add(window), registry.callers["scope-a"].nextOfferAt,
		"a charge has no end to wait for, so holding a bomb does not hold back every other box")
}

func TestAHandWithRoomForNothingIsOfferedNothingUntilItSpends(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")
	roomOf(registry).Fill(accountOf("scope-a"), true)

	for range 5 {
		clock.Advance(window + time.Second)
		clicked(registry, "scope-a")
		registry.sweep(t.Context())
		require.Nil(t, offered(t, events), "a box would pay nothing")
	}

	roomOf(registry).Fill(accountOf("scope-a"), false)

	waitOut(t, registry, clock)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.NotNil(t, offered(t, events), "a charge spent makes room for a box again")
}

func TestAFrozenSetOffersNothingAndAsksNoRoom(t *testing.T) {
	registry, clock := newRuledRegistry(func(rules *Rules) { rules.Frozen = true })
	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)

	assert.Nil(t, offered(t, events), "a frozen set offers no box")
	assert.Empty(t, roomOf(registry).asked, "nothing is offered, so the wallet is asked nothing")

	rulesOf(registry).Frozen = false
	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.Nil(t, offered(t, events), "the slot due while frozen was lost, not banked")

	waitOut(t, registry, clock)
	assert.NotNil(t, offered(t, events), "the next wait brings a box")
}

func TestAClickHeardLongAfterItWasTriedIsNoSignOfAPlayer(t *testing.T) {
	registry, clock := newTestRegistry()
	events := attend(t, registry, "scope-a")

	registry.Clicked(Click{
		Caller: Caller{Scope: "scope-a", Account: accountOf("scope-a")},
		At:     clock.Now().Add(-rulesOf(registry).ActiveWithin - time.Second),
	})
	waitOut(t, registry, clock)

	assert.Nil(t, offered(t, events), "a click told late after a boot is not playing")
}

func TestAClickIsTimedWhenItWasTried(t *testing.T) {
	registry, clock := newTestRegistry()
	events := attend(t, registry, "scope-a")

	tried := clock.Now()
	clock.Advance(rulesOf(registry).ActiveWithin - time.Second)
	registry.Clicked(Click{Caller: Caller{Scope: "scope-a", Account: accountOf("scope-a")}, At: tried})
	clock.Advance(2 * time.Second)
	registry.sweep(t.Context())

	assert.Nil(t, offered(t, events), "the click was tried longer than activeWithin ago")
}

func TestAFullHandOfAnAccountThatClicksFromTheScopeIsOfferedNothing(t *testing.T) {
	registry, clock := newTestRegistry()
	account := Account("acc-1")

	events := attend(t, registry, "scope-a")
	registry.Clicked(Click{Caller: Caller{Scope: "scope-a", Account: account}})
	roomOf(registry).Fill(account, true)

	waitOut(t, registry, clock)
	assert.Nil(t, offered(t, events), "the charges are the account's, whatever address it plays from")
}

func TestAPlayerWhoStoppedClickingNoLongerHoldsBackABox(t *testing.T) {
	registry, clock := newTestRegistry()
	gone := Account("gone")

	events := attend(t, registry, "scope-a")
	registry.Clicked(Click{Caller: Caller{Scope: "scope-a", Account: gone}})
	roomOf(registry).Fill(gone, true)

	clock.Advance(6 * time.Minute)
	clicked(registry, "scope-a")

	waitOut(t, registry, clock)
	clicked(registry, "scope-a")
	registry.sweep(t.Context())
	assert.NotNil(t, offered(t, events), "a full hand of somebody who left is not this player's")
}

func TestACallerWithNoAccountIsOfferedNothing(t *testing.T) {
	registry, clock := newTestRegistry()
	events := attend(t, registry, "scope-a")

	for range 5 {
		clock.Advance(window + time.Second)
		registry.Clicked(Click{Caller: Caller{Scope: "scope-a"}})
		registry.sweep(t.Context())

		require.Nil(t, offered(t, events), "a box pays a charge, and only an account can hold one")
	}
}

func TestTheHourlyCapCountsOnlyTheBoxesCaughtInTheLastHour(t *testing.T) {
	registry, clock := newTestRegistry()
	entry := registry.caller("scope-a", *rulesOf(registry), clock.Now())

	entry.caught = append(entry.caught, clock.Now().Add(-2*time.Hour), clock.Now().Add(-time.Minute), clock.Now())

	assert.Equal(t, 2, registry.caughtWithinTheHour(entry, clock.Now()))
	assert.Len(t, entry.caught, 2, "an older catch is forgotten on the way")
}

func TestACatchIsReportedWithHowLongItTook(t *testing.T) {
	registry, clock := newTestRegistry()

	var (
		caughtBy string
		after    time.Duration
	)
	registry.Observe(Report{Caught: func(scope string, took time.Duration) { caughtBy, after = scope, took }})

	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	clock.Advance(1200 * time.Millisecond)
	_, claimed := registry.Claim(offer.Token, "scope-a", "scope-a")
	require.True(t, claimed)

	assert.Equal(t, "scope-a", caughtBy)
	assert.Equal(t, 1200*time.Millisecond, after, "from the offer being sent, not from the sweep's window")
}

func TestARefusedClaimIsNotACatch(t *testing.T) {
	registry, clock := newTestRegistry()

	caught := 0
	registry.Observe(Report{Caught: func(string, time.Duration) { caught++ }})

	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, stolen := registry.Claim(offer.Token, "scope-b", "scope-b")
	require.False(t, stolen)

	assert.Zero(t, caught)
}

func TestALapsedBoxIsReportedAgainstItsCaller(t *testing.T) {
	registry, clock := newTestRegistry()

	var missedBy []string
	registry.Observe(Report{Lapsed: func(scope string) { missedBy = append(missedBy, scope) }})

	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	require.NotNil(t, offered(t, events))

	clock.Advance(16 * time.Second)
	registry.sweep(t.Context())

	assert.Equal(t, []string{"scope-a"}, missedBy)
}

func foreign(registry *Registry) map[string]int {
	claims := map[string]int{}
	registry.Observe(Report{Foreign: func(scope string) { claims[scope]++ }})
	return claims
}

func TestClaimingABoxOfferedToAnotherCallerIsReported(t *testing.T) {
	registry, clock := newTestRegistry()
	claims := foreign(registry)
	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, stolen := registry.Claim(offer.Token, "scope-b", "scope-b")
	_, mine := registry.Claim(offer.Token, "scope-a", "scope-a")
	_, late := registry.Claim(offer.Token, "scope-c", "scope-c")

	assert.False(t, stolen)
	assert.True(t, mine)
	assert.False(t, late)
	assert.Equal(t, map[string]int{"scope-b": 1, "scope-c": 1}, claims, "the owner's own claim is no evidence, before or after the others")
}

func TestClaimingATokenNeverOfferedIsReported(t *testing.T) {
	registry, _ := newTestRegistry()
	claims := foreign(registry)

	_, claimed := registry.Claim("not-a-token", "scope-a", "scope-a")

	assert.False(t, claimed)
	assert.Equal(t, map[string]int{"scope-a": 1}, claims)
}

func TestClaimingYourOwnBoxTwiceOrLateIsNotReported(t *testing.T) {
	registry, clock := newTestRegistry()
	claims := foreign(registry)
	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	first := offered(t, events)
	require.NotNil(t, first)

	_, once := registry.Claim(first.Token, "scope-a", "scope-a")
	_, twice := registry.Claim(first.Token, "scope-a", "scope-a")
	require.True(t, once)
	require.False(t, twice)

	waitOut(t, registry, clock)
	second := offered(t, events)
	require.NotNil(t, second)

	clock.Advance(16 * time.Second)
	_, beforeTheSweep := registry.Claim(second.Token, "scope-a", "scope-a")
	registry.sweep(t.Context())
	_, afterTheSweep := registry.Claim(second.Token, "scope-a", "scope-a")

	assert.False(t, beforeTheSweep)
	assert.False(t, afterTheSweep)
	assert.Empty(t, claims)
}

func TestASpentTokenIsForgottenAfterAWhile(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)
	_, claimed := registry.Claim(offer.Token, "scope-a", "scope-a")
	require.True(t, claimed)

	clock.Advance(rememberSpent + time.Second)
	registry.sweep(t.Context())

	assert.Empty(t, registry.spent)
}

func TestLapsedTokensAreForgottenRatherThanKept(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	waitOut(t, registry, clock)
	require.NotNil(t, offered(t, events))

	clock.Advance(16 * time.Second)
	registry.sweep(t.Context())

	assert.Empty(t, registry.offers)
}

func TestEveryTokenIsDifferent(t *testing.T) {
	registry, clock := newTestRegistry()
	events := playing(t, registry, "scope-a")

	seen := cpcolls.NewSet[string]()
	for range 20 {
		clock.Advance(window + time.Second)
		clicked(registry, "scope-a")
		registry.sweep(t.Context())

		offer := offered(t, events)
		require.NotNil(t, offer)
		require.False(t, seen.Contains(offer.Token), "a token was handed out twice")
		seen.Add(offer.Token)

		clock.Advance(16 * time.Second)
		registry.sweep(t.Context())
	}
}

func TestACallerThatIsNotReadingIsDroppedRatherThanBlocking(t *testing.T) {
	registry, _ := newTestRegistry()
	events := playing(t, registry, "scope-a")
	entry := registry.callers["scope-a"]

	for range offerBuffer * 4 {
		entry.send(Offer{Token: "a-token"})
	}

	assert.Len(t, events, offerBuffer)
}
