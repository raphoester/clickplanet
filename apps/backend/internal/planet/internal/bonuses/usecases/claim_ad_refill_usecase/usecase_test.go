package claim_ad_refill_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_ad_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type onePrice struct{}

func (onePrice) Price(string) clicks.Price { return clicks.Price{Slowdown: 1} }

var buckets = clicks.NewBuckets(func() clicks.Throttle {
	return clicks.Throttle{
		Tempo: 1, PerSecond: 0.2, Burst: 60,
		ScopeMultiplier: 10, V6ScopeMultiplier: 10, GuestScopeMultiplier: 1, LinkedMultiplier: 2,
		NewAccountClicks: 60,
	}
})

type fixture struct {
	limiter *cpratelimit.Limiter
	useCase *claim_ad_refill_usecase.UseCase
}

func setUp(enabled bool) fixture {
	rules := bonuses.TestRules()
	rules.AdRefill.Enabled = enabled
	inForce := func() bonuses.Rules { return rules }
	clock := cptime.NewFixedClock(time.Date(2026, 10, 30, 12, 34, 0, 0, time.UTC))
	limiter := cpratelimit.New("test", cpratelimit.Config{PerSecond: 0.2, Burst: 60}, clock)

	return fixture{
		limiter: limiter,
		useCase: claim_ad_refill_usecase.New(cpatomic.FakeTransactioner{},
			bonuses.NewAdRefills(bonuses.NewMemoryGrants(), inForce, clock),
			clicks.NewRefilling(limiter, buckets), onePrice{}, inForce),
	}
}

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), ada)
}

func (f fixture) spend(t *testing.T, n int) {
	t.Helper()

	keys := buckets.Keys(clicks.Payer{Scope: "1.2.3.4", Account: ada}, clicks.Price{Slowdown: 1})
	for range n {
		allowed, _ := f.limiter.TakeAll(1, keys...)
		require.True(t, allowed)
	}
}

func (f fixture) tokens() float64 {
	return f.limiter.Peek(buckets.Bank(clicks.Payer{Scope: "1.2.3.4", Account: ada})[0]).Tokens
}

func TestAnAdFillsTheBank(t *testing.T) {
	f := setUp(true)
	f.spend(t, 50)

	budget, err := f.useCase.Execute(played(t), claim_ad_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.InDelta(t, 60, budget.Tokens, 1e-9)
	assert.Equal(t, 60, budget.Capacity)
}

func TestASecondAdInTheSpanIsRefusedWithWhenTheNextMayBe(t *testing.T) {
	f := setUp(true)
	f.spend(t, 50)
	_, err := f.useCase.Execute(played(t), claim_ad_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err)
	f.spend(t, 50)

	_, err = f.useCase.Execute(played(t), claim_ad_refill_usecase.In{CountryID: "fr"})

	assert.Equal(t, bonuses.AdRefillRefusal{Until: time.Date(2026, 10, 30, 13, 0, 0, 0, time.UTC)}, err)
	assert.InDelta(t, 10, f.tokens(), 1e-9, "nothing filled")
}

func TestAFullBankIsRefusedAndRecordsNothing(t *testing.T) {
	f := setUp(true)

	_, err := f.useCase.Execute(played(t), claim_ad_refill_usecase.In{CountryID: "fr"})
	require.ErrorIs(t, err, clicks.ErrBankFull)

	f.spend(t, 50)
	_, err = f.useCase.Execute(played(t), claim_ad_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err, "an ad watched at a full meter wastes nothing")
}

func TestNoAdFillsTheBankWhileTheRuleIsOff(t *testing.T) {
	f := setUp(false)
	f.spend(t, 50)

	_, err := f.useCase.Execute(played(t), claim_ad_refill_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, claim_ad_refill_usecase.ErrAdRefillOff)
	assert.InDelta(t, 10, f.tokens(), 1e-9)
}

func TestACallerWithNoAccountHasNoAdRefill(t *testing.T) {
	f := setUp(true)

	_, err := f.useCase.Execute(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), claim_ad_refill_usecase.In{})

	require.ErrorIs(t, err, claim_ad_refill_usecase.ErrNoAccount)
}
