package use_refill_usecase_test

import (
	"context"
	"errors"
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

const guest bonuses.Holder = "a-guest"

type onePrice struct{}

func (onePrice) Price(string) clicks.Price { return clicks.Price{Slowdown: 1} }

var config = clicks.ThrottleConfig{Config: cpratelimit.Config{PerSecond: 0.2, Burst: 60}, ScopeMultiplier: 10}

type fixture struct {
	store   *bonuses.MemoryStore
	limiter *cpratelimit.Limiter
	useCase *use_refill_usecase.UseCase
}

func setup(t *testing.T, held bool) fixture {
	t.Helper()

	store := bonuses.NewMemoryStore()
	refills := bonuses.NewCharges(store, bonuses.ChargesConfig{})
	if held {
		require.NoError(t, refills.Grant(t.Context(), guest, bonuses.KindRefill, 1))
	}
	limiter := cpratelimit.New("test", config.Config, cptime.NewFixedClock(epoch))

	return fixture{
		store:   store,
		limiter: limiter,
		useCase: use_refill_usecase.New(refills, limiter, onePrice{}, config.Buckets()),
	}
}

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")
}

func (f fixture) spend(t *testing.T, account string, n int) {
	t.Helper()

	keys := config.Buckets().Keys(clicks.Payer{Scope: "1.2.3.4", Account: account}, clicks.Price{Slowdown: 1})
	for range n {
		allowed, _ := f.limiter.TakeAll(1, keys...)
		require.True(t, allowed)
	}
}

func TestARefillFillsTheBankAndIsSpent(t *testing.T) {
	f := setup(t, true)
	f.spend(t, "a-guest", 50)

	out, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.InDelta(t, 60, out.Budget.Tokens, 1e-9, "the bank is full")
	assert.Equal(t, 60, out.Budget.Capacity, "and no bigger")
	assert.Equal(t, bonuses.Held{}, out.Held)
	assert.Equal(t, bonuses.Held{}, f.store.Kept(guest), "the refill is spent")
}

func TestAFullBankIsRefusedAndSpendsNothing(t *testing.T) {
	f := setup(t, true)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, use_refill_usecase.ErrBankFull)
	assert.Equal(t, bonuses.Held{Refill: true}, f.store.Kept(guest), "a press on a full meter wastes nothing")
}

func TestNoRefillFillsNothing(t *testing.T) {
	f := setup(t, false)
	f.spend(t, "a-guest", 50)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, use_refill_usecase.ErrNoRefill)
	state := f.limiter.Peek(config.Buckets().Bank(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"})[0])
	assert.InDelta(t, 10, state.Tokens, 1e-9)
}

func TestACallerWithNoAccountHasNoRefill(t *testing.T) {
	f := setup(t, true)

	_, err := f.useCase.Execute(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), use_refill_usecase.In{})

	require.ErrorIs(t, err, use_refill_usecase.ErrNoRefill)
	assert.Equal(t, bonuses.Held{Refill: true}, f.store.Kept(guest))
}

func TestTheScopesBucketIsNotFilled(t *testing.T) {
	f := setup(t, true)
	f.spend(t, "a-guest", 50)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	scope := f.limiter.Peek(config.Buckets().Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}, clicks.Price{})[2])
	assert.InDelta(t, 550, scope.Tokens, 1e-9, "the scope's bucket, shared by everyone behind the address, keeps its level")
}

func TestARefillFillsTheBucketTheScopesGuestsShare(t *testing.T) {
	f := setup(t, true)
	f.spend(t, "another-guest", 50)

	out, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err, "its own bucket is full, but the guests' it spends from is not")

	assert.InDelta(t, 60, out.Budget.Tokens, 1e-9)
	assert.Equal(t, clicks.SharedWithNobody, out.Budget.SharedWith)
}

func TestAStoreThatFailsFillsNothing(t *testing.T) {
	f := setup(t, true)
	f.spend(t, "a-guest", 50)
	down := errors.New("postgres is down")
	f.store.FailWith(down)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, use_refill_usecase.ErrNoRefill, "the player is not told it holds no refill")
	state := f.limiter.Peek(config.Buckets().Bank(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"})[0])
	assert.InDelta(t, 10, state.Tokens, 1e-9)
}
