package claim_bonus_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

type stubRegistry struct {
	reward    bonuses.Reward
	claimable bool

	token     string
	scope     string
	published []bonuses.Taken
}

func (s *stubRegistry) Claim(token string, scope string) (bonuses.Reward, bool) {
	s.token, s.scope = token, scope
	if !s.claimable {
		return bonuses.Reward{}, false
	}

	return s.reward, true
}

func (s *stubRegistry) Publish(taken bonuses.Taken) { s.published = append(s.published, taken) }

func (s *stubRegistry) Multiplier() float64 { return 3 }

type stubBooster struct {
	key        string
	multiplier float64
	until      time.Time
	state      cpratelimit.State
	peeked     []cpratelimit.Key
}

func (s *stubBooster) Peek(key cpratelimit.Key) cpratelimit.State {
	s.peeked = append(s.peeked, key)
	return s.state
}

func (s *stubBooster) Boost(key string, multiplier float64, until time.Time) cpratelimit.State {
	s.key, s.multiplier, s.until = key, multiplier, until
	return s.state
}

// stubCharger records the charges granted, and answers what they add up to.
type stubCharger struct {
	granted []grantedCharge
}

type grantedCharge struct {
	holder bonuses.Holder
	kind   bonuses.Kind
}

func (s *stubCharger) Grant(holder bonuses.Holder, kind bonuses.Kind) {
	s.granted = append(s.granted, grantedCharge{holder: holder, kind: kind})
}

func (s *stubCharger) Held(holder bonuses.Holder) bonuses.Held {
	var held bonuses.Held
	for _, g := range s.granted {
		if g.holder != holder {
			continue
		}
		switch g.kind {
		case bonuses.KindBomb:
			held.Bomb = true
		case bonuses.KindEncloseClicks:
			held.Enclose = true
		case bonuses.KindSpreadClicks:
			held.SpreadClicks = 8
		case bonuses.KindTripleClicks:
		}
	}

	return held
}

type stubPricer clicks.Price

func (p stubPricer) Price(string) clicks.Price { return clicks.Price(p) }

var onePrice = stubPricer{Cost: 1}

var buckets = clicks.ThrottleConfig{}.Buckets()

func granted() *stubRegistry {
	return &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindTripleClicks, Duration: time.Minute}}
}

func TestAClaimStartsTheBoostForTheDurationGranted(t *testing.T) {
	booster := &stubBooster{state: cpratelimit.State{Capacity: 30, PerSecond: 3}}

	out, err := claim_bonus_usecase.New(granted(), booster, onePrice, &stubCharger{}, buckets, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus_usecase.In{Token: "a-token", CountryID: "fr"})
	require.NoError(t, err)

	assert.InDelta(t, 3.0, booster.multiplier, 1e-9)
	assert.Equal(t, epoch.Add(time.Minute), booster.until)
	assert.Equal(t, 30, out.Budget.Capacity)
	assert.Equal(t, time.Minute, out.Duration)
}

func TestTheWidenedAllowanceIsPricedForTheCatchersCountry(t *testing.T) {
	booster := &stubBooster{state: cpratelimit.State{Tokens: 12, Capacity: 30, PerSecond: 3}}

	out, err := claim_bonus_usecase.New(granted(), booster, stubPricer{Cost: 3, Share: 0.4}, &stubCharger{}, buckets, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus_usecase.In{Token: "a-token", CountryID: "bg"})
	require.NoError(t, err)

	assert.Equal(t, 10, out.Budget.Capacity, "a triple bonus at a cost of three is ten clicks")
	assert.InDelta(t, 1.0, out.Budget.PerSecond, 1e-9)
	assert.InDelta(t, 4.0, out.Budget.Tokens, 1e-9)
	assert.InDelta(t, 3, out.Budget.Price.Cost, 1e-9)
}

func TestTheClaimAndTheBoostUseTheSameScope(t *testing.T) {
	registry, booster := granted(), &stubBooster{}

	_, err := claim_bonus_usecase.New(registry, booster, onePrice, &stubCharger{}, buckets, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus_usecase.In{Token: "a-token"})
	require.NoError(t, err)

	assert.Equal(t, registry.scope, booster.key)
	assert.Equal(t, cpctx.RateLimitKey(t.Context()), booster.key)
}

func TestATripleBoostsTheAccountAndNeverTheScope(t *testing.T) {
	registry, booster := granted(), &stubBooster{}
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	_, err := claim_bonus_usecase.New(registry, booster, onePrice, &stubCharger{}, buckets, cptime.NewFixedClock(epoch)).
		Execute(ctx, claim_bonus_usecase.In{Token: "a-token"})
	require.NoError(t, err)

	assert.Equal(t, "1.2.3.4", registry.scope, "the offer is still the scope's")
	assert.Equal(t, "account:a-guest", booster.key)
	assert.Equal(t, buckets.Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}), booster.peeked,
		"the answer is read off both buckets")
}

func TestACatchIsAnnouncedWithTheCountryTheClaimNamed(t *testing.T) {
	registry := granted()

	_, err := claim_bonus_usecase.New(registry, &stubBooster{}, onePrice, &stubCharger{}, buckets, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus_usecase.In{Token: "a-token", CountryID: "jp"})
	require.NoError(t, err)

	require.Len(t, registry.published, 1)
	assert.Equal(t, "jp", registry.published[0].CountryID)
}

func TestARefusedClaimBoostsNothingAndAnnouncesNothing(t *testing.T) {
	registry, booster := &stubRegistry{claimable: false}, &stubBooster{}

	_, err := claim_bonus_usecase.New(registry, booster, onePrice, &stubCharger{}, buckets, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus_usecase.In{Token: "not-mine"})

	require.ErrorIs(t, err, claim_bonus_usecase.ErrNoSuchBonus)
	assert.Zero(t, booster.multiplier)
	assert.Empty(t, registry.published)
}

func TestABombClaimHandsOverTheBomb(t *testing.T) {
	registry := &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindBomb}}
	booster := &stubBooster{state: cpratelimit.State{Capacity: 10, PerSecond: 1}}
	charger := &stubCharger{}

	out, err := claim_bonus_usecase.New(registry, booster, onePrice, charger, buckets, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus_usecase.In{Token: "a-token", CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, []grantedCharge{{holder: bonuses.Holder("scope:" + cpctx.RateLimitKey(t.Context())), kind: bonuses.KindBomb}}, charger.granted)
	assert.Zero(t, booster.multiplier, "a bomb is not a boost")
	assert.Equal(t, bonuses.KindBomb, out.Kind)
	assert.Zero(t, out.Duration, "a bomb is kept until it is dropped")
	assert.Equal(t, bonuses.Held{Bomb: true}, out.Held)
}

func TestAChargeIsTheAccountsAndNotTheAddresss(t *testing.T) {
	registry := &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindBomb}}
	charger := &stubCharger{}
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	_, err := claim_bonus_usecase.New(registry, &stubBooster{}, onePrice, charger, buckets, cptime.NewFixedClock(epoch)).
		Execute(ctx, claim_bonus_usecase.In{Token: "a-token"})
	require.NoError(t, err)

	assert.Equal(t, "1.2.3.4", registry.scope, "the offer is still the scope's")
	assert.Equal(t, []grantedCharge{{holder: "account:a-guest", kind: bonuses.KindBomb}}, charger.granted)
}

func TestATripleGrantsNoCharge(t *testing.T) {
	charger := &stubCharger{}

	out, err := claim_bonus_usecase.New(granted(), &stubBooster{}, onePrice, charger, buckets, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus_usecase.In{Token: "a-token"})
	require.NoError(t, err)

	assert.Empty(t, charger.granted)
	assert.Equal(t, bonuses.Held{}, out.Held)
}

func TestASpreadClaimHandsOverTheChargeAndWidensNothing(t *testing.T) {
	registry := &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindSpreadClicks}}
	booster := &stubBooster{state: cpratelimit.State{Tokens: 4, Capacity: 10, PerSecond: 1}}
	charger := &stubCharger{}

	out, err := claim_bonus_usecase.New(registry, booster, onePrice, charger, buckets, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus_usecase.In{Token: "a-token", CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, []grantedCharge{{holder: bonuses.Holder("scope:" + cpctx.RateLimitKey(t.Context())), kind: bonuses.KindSpreadClicks}}, charger.granted)
	assert.Zero(t, booster.multiplier, "a spread is not a boost")
	assert.Equal(t, 10, out.Budget.Capacity, "the allowance is answered as it stands")
	assert.Equal(t, bonuses.KindSpreadClicks, out.Kind)
	assert.Equal(t, bonuses.Held{SpreadClicks: 8}, out.Held)
	require.Len(t, registry.published, 1)
}

func TestAnEncloseClaimHandsOverTheChargeAndWidensNothing(t *testing.T) {
	registry := &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindEncloseClicks}}
	booster := &stubBooster{state: cpratelimit.State{Tokens: 4, Capacity: 10, PerSecond: 1}}
	charger := &stubCharger{}

	out, err := claim_bonus_usecase.New(registry, booster, onePrice, charger, buckets, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus_usecase.In{Token: "a-token", CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, []grantedCharge{{holder: bonuses.Holder("scope:" + cpctx.RateLimitKey(t.Context())), kind: bonuses.KindEncloseClicks}}, charger.granted)
	assert.Zero(t, booster.multiplier, "an enclose is not a boost")
	assert.Equal(t, 10, out.Budget.Capacity, "the allowance is answered as it stands")
	assert.Equal(t, bonuses.Held{Enclose: true}, out.Held)
	require.Len(t, registry.published, 1)
}
