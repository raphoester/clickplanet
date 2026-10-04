package bonuses

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_allegiance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	bigFlag   = "pl"
	smallFlag = "ad"
)

var (
	everyWeight = map[Kind]float64{KindRefill: 1, KindSpreadClicks: 1, KindEncloseClicks: 1, KindBomb: 1}
	refillsOnly = map[Kind]float64{KindRefill: 1}
)

func TestWithNoBandsEveryShareDrawsFromKinds(t *testing.T) {
	bands := Config{Kinds: everyWeight}.Bands()

	assert.Equal(t, KindBand{Share: 0, Kinds: everyWeight}, bands.At(0))
	assert.Equal(t, KindBand{Share: 0, Kinds: everyWeight}, bands.At(0.9))
}

func TestABandReplacesTheKindsFromItsShareUp(t *testing.T) {
	noBomb := map[Kind]float64{KindRefill: 5, KindSpreadClicks: 2}
	bands := Config{
		Kinds:        everyWeight,
		KindsByShare: []KindBand{{Share: 0.10, Kinds: noBomb}, {Share: 0.25, Kinds: refillsOnly}},
	}.Bands()

	assert.Equal(t, everyWeight, bands.At(0).Kinds)
	assert.Equal(t, everyWeight, bands.At(0.0999).Kinds)
	assert.Equal(t, noBomb, bands.At(0.10).Kinds)
	assert.Equal(t, noBomb, bands.At(0.2).Kinds)
	assert.Equal(t, refillsOnly, bands.At(0.25).Kinds)
	assert.InDelta(t, 0.25, bands.At(1).Share, 1e-9)
}

func TestTheBandsStartFromTheDefaultKinds(t *testing.T) {
	bands := Config{KindsByShare: []KindBand{{Share: 0.2, Kinds: refillsOnly}}}.Bands()

	assert.Equal(t, defaultKinds(), bands.At(0).Kinds)
	assert.Equal(t, refillsOnly, bands.At(0.3).Kinds)
}

func TestBandsThatMakeNoSenseRefuseTheConfig(t *testing.T) {
	banded := func(bands ...KindBand) Config { return Config{KindsByShare: bands} }

	require.NoError(t, banded(KindBand{Share: 0.1, Kinds: refillsOnly}, KindBand{Share: 0.2, Kinds: refillsOnly}).Validate())

	for _, share := range []float64{0, -0.1, 1, math.NaN()} {
		assert.ErrorContains(t, banded(KindBand{Share: share, Kinds: refillsOnly}).Validate(), "kindsByShare[0].share")
	}
	assert.ErrorContains(t, banded(KindBand{Share: 0.2, Kinds: refillsOnly}, KindBand{Share: 0.1, Kinds: refillsOnly}).Validate(),
		"kindsByShare[1].share")
	assert.ErrorContains(t, banded(KindBand{Share: 0.1, Kinds: refillsOnly}, KindBand{Share: 0.1, Kinds: refillsOnly}).Validate(),
		"kindsByShare[1].share")

	assert.ErrorContains(t, banded(KindBand{Share: 0.1, Kinds: map[Kind]float64{"quadruple_clicks": 1}}).Validate(), "quadruple_clicks")
	assert.ErrorContains(t, banded(KindBand{Share: 0.1, Kinds: map[Kind]float64{KindBomb: -1}}).Validate(), "kindsByShare[0].kinds.bomb")
	assert.ErrorContains(t, banded(KindBand{Share: 0.1, Kinds: map[Kind]float64{KindBomb: 0}}).Validate(), "no kind a weight")
	assert.ErrorContains(t, banded(KindBand{Share: 0.1}).Validate(), "no kind a weight", "an empty band offers nothing")
}

// newBandedRegistry draws from base below 20% of the map and offers only refills from it.
func newBandedRegistry(base map[Kind]float64) (*Registry, *cptime.FixedClock) {
	clock := cptime.NewFixedClock(epoch)

	return New(Config{
		MinInterval:       window,
		MaxInterval:       window,
		MissRetry:         20 * time.Second,
		OfferTTL:          15 * time.Second,
		Kinds:             base,
		KindsByShare:      []KindBand{{Share: 0.2, Kinds: refillsOnly}},
		ActiveWithin:      5 * time.Minute,
		ForgetAfter:       5 * time.Minute,
		MaxChargesPerHour: 6,
		SweepInterval:     time.Second,
	}, clock, newFakeHoldings(), fakeShares{bigFlag: 0.4, smallFlag: 0.01}, inmemory_allegiance_store.New()), clock
}

// plays is a click from scope by account, whose main flag is country.
func plays(t *testing.T, r *Registry, scope string, account string, country string) {
	t.Helper()

	paint(t, r, clicks.AccountAllegianceKey(account), country)
	r.Clicked(Entrant(scope), scope, Holder(account))
}

// offerableTo is what a sweep would offer scope now, its flags read as a sweep reads them.
func offerableTo(t *testing.T, r *Registry, scope string, clock *cptime.FixedClock) *cpcolls.Set[Kind] {
	t.Helper()

	entry := r.callers[Entrant(scope)]
	r.forgetIdlePlayers(entry, clock.Now())
	flags, err := r.readFlags(t.Context(), map[Entrant][]clicks.AllegianceKey{Entrant(scope): keysOf(entry)})
	require.NoError(t, err)

	band, read := r.band(flags, entry)
	require.True(t, read)
	return r.offerable(entry, clock.Now(), band)
}

func TestABigCountrysPlayerIsOfferedOnlyRefills(t *testing.T) {
	registry, clock := newBandedRegistry(everyWeight)
	events := attend(t, registry, "scope-a")
	plays(t, registry, "scope-a", "acc-pl", bigFlag)

	assert.Equal(t, cpcolls.NewSet(KindRefill), offerableTo(t, registry, "scope-a", clock))

	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)
	assert.Equal(t, KindRefill, offer.Kind)
}

func TestASmallCountrysPlayerDrawsFromTheWholeTable(t *testing.T) {
	registry, clock := newBandedRegistry(everyWeight)
	attend(t, registry, "scope-a")
	plays(t, registry, "scope-a", "acc-ad", smallFlag)

	assert.Equal(t, cpcolls.NewSet(Kinds...), offerableTo(t, registry, "scope-a", clock))
}

func TestAnAccountBringsItsFlagToANewAddress(t *testing.T) {
	registry, clock := newBandedRegistry(everyWeight)
	attend(t, registry, "scope-b")
	paint(t, registry, clicks.ScopeAllegianceKey("scope-b"), smallFlag)
	plays(t, registry, "scope-b", "acc-pl", bigFlag)

	assert.Equal(t, cpcolls.NewSet(KindRefill), offerableTo(t, registry, "scope-b", clock))
}

func TestAFreshAccountOnAnAddressKeepsTheAddresssFlag(t *testing.T) {
	registry, clock := newBandedRegistry(everyWeight)
	attend(t, registry, "scope-a")
	paint(t, registry, clicks.ScopeAllegianceKey("scope-a"), bigFlag)
	plays(t, registry, "scope-a", "a-fresh-guest", smallFlag)

	assert.Equal(t, cpcolls.NewSet(KindRefill), offerableTo(t, registry, "scope-a", clock),
		"a second account opened on the same address is still that address's player")
}

func TestAScopeDrawsTheBandOfTheBiggestFlagPlayingFromIt(t *testing.T) {
	registry, clock := newBandedRegistry(everyWeight)
	attend(t, registry, "scope-a")
	paint(t, registry, clicks.ScopeAllegianceKey("scope-a"), smallFlag)
	plays(t, registry, "scope-a", "acc-ad", smallFlag)
	plays(t, registry, "scope-a", "acc-pl", bigFlag)

	assert.Equal(t, cpcolls.NewSet(KindRefill), offerableTo(t, registry, "scope-a", clock),
		"every account on the scope sees the box, so a big country's player could claim it")

	clock.Advance(6 * time.Minute)
	plays(t, registry, "scope-a", "acc-ad", smallFlag)

	assert.Equal(t, cpcolls.NewSet(Kinds...), offerableTo(t, registry, "scope-a", clock),
		"once the big country's player stops, the scope's own flag is the small one")
}

func TestAKindHeldIsStillLeftOutOfTheBand(t *testing.T) {
	registry, clock := newBandedRegistry(everyWeight)
	events := attend(t, registry, "scope-a")
	plays(t, registry, "scope-a", "acc-pl", bigFlag)
	holdingsOf(registry).grant("acc-pl", KindRefill)

	assert.True(t, offerableTo(t, registry, "scope-a", clock).Empty())

	waitOut(registry, clock)
	assert.Nil(t, offered(t, events), "a band with nothing left to offer loses the slot")
}

func TestAnOfferIsReportedWithItsKindAndBand(t *testing.T) {
	registry, clock := newBandedRegistry(everyWeight)

	type report struct {
		kind Kind
		band float64
	}
	var offers []report
	registry.Observe(Report{Offered: func(kind Kind, band float64) { offers = append(offers, report{kind, band}) }})

	attend(t, registry, "scope-a")
	plays(t, registry, "scope-a", "acc-pl", bigFlag)
	waitOut(registry, clock)

	assert.Equal(t, []report{{KindRefill, 0.2}}, offers)
}

func TestAQuizIsWorthWhatThePlayersBandOffers(t *testing.T) {
	for country, want := range map[string]Kind{smallFlag: KindBomb, bigFlag: KindRefill} {
		registry, clock := newBandedRegistry(map[Kind]float64{KindBomb: 1})
		registry.Quizzing(quizzes.Config{
			Enabled: true, MinInterval: quizWindow, MaxInterval: quizWindow,
			OfferTTL: bannerFor, AnswerWindow: answerIn, MaxChargesPerHour: 6,
		}, fixedBank{})

		events := attend(t, registry, "scope-a")
		plays(t, registry, "scope-a", "acc", country)
		waitOutQuiz(registry, clock)

		offer := quizOffered(t, events)
		require.NotNil(t, offer)
		asked, ok := registry.OpenQuiz(offer.Token, "scope-a")
		require.True(t, ok)

		answered, ok := registry.AnswerQuiz(offer.Token, "scope-a", indexOf(asked.Options, rightAnswer))
		require.True(t, ok)
		assert.Equal(t, want, answered.Reward.Kind, "a quiz for a player of %s", country)
	}
}

func TestNothingIsOfferedWhileTheFlagsCannotBeReadAndTheSlotIsKept(t *testing.T) {
	registry, clock := newBandedRegistry(everyWeight)
	events := attend(t, registry, "scope-a")
	plays(t, registry, "scope-a", "acc-ad", smallFlag)
	flagsOf(registry).FailWith(errors.New("postgres is down"))

	waitOut(registry, clock)
	require.Nil(t, offered(t, events), "no band, so no box")

	flagsOf(registry).Heal()
	clock.Advance(time.Second)
	registry.sweep(t.Context())
	assert.NotNil(t, offered(t, events), "the caller stayed due, so the next sweep offers")
}

func TestACallerWhosePlayersFlagWasNotReadWaitsForTheNextSweep(t *testing.T) {
	registry, _ := newBandedRegistry(everyWeight)
	plays(t, registry, "scope-a", "acc-ad", smallFlag)
	entry := registry.callers["scope-a"]

	read := map[clicks.AllegianceKey]clicks.Allegiance{clicks.ScopeAllegianceKey("scope-a"): {}}
	_, ok := registry.band(read, entry)
	assert.False(t, ok, "a big country's player who joined after the read could be left out of the band")

	read[clicks.AccountAllegianceKey("acc-ad")] = clicks.Allegiance{}
	_, ok = registry.band(read, entry)
	assert.True(t, ok)
}

func TestASignedInPlayersBandReadsItsOwnFlagAndItsNetworks(t *testing.T) {
	registry, clock := newBandedRegistry(everyWeight)
	attend(t, registry, "account:acc-ad")
	paint(t, registry, clicks.AccountAllegianceKey("acc-ad"), smallFlag)
	registry.Clicked("account:acc-ad", "scope-a", "acc-ad")

	assert.Equal(t, cpcolls.NewSet(Kinds...), offerableTo(t, registry, "account:acc-ad", clock),
		"its own box, drawn from its own small flag")

	paint(t, registry, clicks.ScopeAllegianceKey("scope-a"), bigFlag)

	assert.Equal(t, cpcolls.NewSet(KindRefill), offerableTo(t, registry, "account:acc-ad", clock),
		"a fresh account on a big country's network is not a way around the band")
}
