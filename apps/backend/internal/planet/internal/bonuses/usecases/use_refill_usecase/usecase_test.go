package use_refill_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/use_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)

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
	refills *bonuses.MemoryStore
	limiter *cpratelimit.Limiter
	useCase *use_refill_usecase.UseCase
}

func setup(held bool) fixture {
	refills := bonuses.NewMemoryStore()
	refills.Keep("a-guest", bonuses.Held{Refill: held})
	limiter := cpratelimit.New("test", cpratelimit.Config{PerSecond: 0.2, Burst: 60}, cptime.NewFixedClock(epoch))

	return fixture{
		refills: refills,
		limiter: limiter,
		useCase: use_refill_usecase.New(bonuses.NewCharges(refills, bonuses.TestRules), clicks.NewRefilling(limiter, buckets), onePrice{}),
	}
}

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")
}

func (f fixture) spend(t *testing.T, account string, n int) {
	t.Helper()

	keys := buckets.Keys(clicks.Payer{Scope: "1.2.3.4", Account: account}, clicks.Price{Slowdown: 1})
	for range n {
		allowed, _ := f.limiter.TakeAll(1, keys...)
		require.True(t, allowed)
	}
}

func TestARefillFillsTheBankAndIsSpent(t *testing.T) {
	f := setup(true)
	f.spend(t, "a-guest", 50)

	out, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.InDelta(t, 60, out.Budget.Tokens, 1e-9, "the bank is full")
	assert.Equal(t, 60, out.Budget.Capacity, "and no bigger")
	assert.Equal(t, bonuses.Held{}, out.Held)
	assert.Equal(t, bonuses.Held{}, f.refills.Kept("a-guest"), "the refill is spent")
}

func TestAFullBankIsRefusedAndSpendsNothing(t *testing.T) {
	f := setup(true)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrBankFull)
	assert.True(t, f.refills.Kept("a-guest").Refill, "a press on a full meter wastes nothing")
}

func TestNoRefillFillsNothing(t *testing.T) {
	f := setup(false)
	f.spend(t, "a-guest", 50)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, use_refill_usecase.ErrNoRefill)
	state := f.limiter.Peek(buckets.Bank(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"})[0])
	assert.InDelta(t, 10, state.Tokens, 1e-9)
}

func TestACallerWithNoAccountHasNoRefill(t *testing.T) {
	f := setup(true)

	_, err := f.useCase.Execute(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), use_refill_usecase.In{})

	require.ErrorIs(t, err, use_refill_usecase.ErrNoRefill)
	assert.True(t, f.refills.Kept("a-guest").Refill)
}

var scope = buckets.Ceiling(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"})[0]

func (f fixture) drainScope(t *testing.T, n float64) {
	t.Helper()

	allowed, _ := f.limiter.TakeAll(n, scope)
	require.True(t, allowed)
}

func TestTheScopesBucketIsGivenWhatTheRefillGave(t *testing.T) {
	f := setup(true)
	f.drainScope(t, 200)
	f.spend(t, "a-guest", 50)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.InDelta(t, 400, f.limiter.Peek(scope).Tokens, 1e-9,
		"50 clicks given, so the scope everyone behind the address shares is no tighter for it, and no looser")
}

func TestARefillFillsWhatTheScopeLacksWhenItIsTheTighter(t *testing.T) {
	f := setup(true)
	f.drainScope(t, 576)

	out, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err, "its own bank is full, but the scope it spends from holds 24")

	assert.InDelta(t, 60, out.Budget.Tokens, 1e-9, "a full bank to spend")
	assert.Equal(t, 60, out.Budget.Capacity, "read against its own bank, not the scope's 600")
	assert.InDelta(t, 60, f.limiter.Peek(scope).Tokens, 1e-9, "the scope is given the 36 the player lacked, no more")
	assert.False(t, f.refills.Kept("a-guest").Refill, "the refill is spent")
}

func TestAFullBankOnAScopeHoldingABankIsRefused(t *testing.T) {
	f := setup(true)
	f.drainScope(t, 540)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrBankFull, "the scope still holds the bank the player can spend")
	assert.InDelta(t, 60, f.limiter.Peek(scope).Tokens, 1e-9)
	assert.True(t, f.refills.Kept("a-guest").Refill)
}

func TestARefillFillsTheBucketTheScopesGuestsShare(t *testing.T) {
	f := setup(true)
	f.spend(t, "another-guest", 50)

	out, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err, "its own bucket is full, but the guests' it spends from is not")

	assert.InDelta(t, 60, out.Budget.Tokens, 1e-9)
	assert.Equal(t, clicks.SharedWithNobody, out.Budget.SharedWith)
}
