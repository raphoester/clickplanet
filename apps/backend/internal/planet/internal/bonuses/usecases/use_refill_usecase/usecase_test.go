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

// stubRefills holds one refill per holder named, and nothing else.
type stubRefills struct {
	held  map[bonuses.Holder]bool
	spent []bonuses.Holder
}

func (s *stubRefills) SpendRefill(holder bonuses.Holder) bool {
	s.spent = append(s.spent, holder)
	held := s.held[holder]
	s.held[holder] = false

	return held
}

func (s *stubRefills) Held(holder bonuses.Holder) bonuses.Held {
	return bonuses.Held{Refill: s.held[holder]}
}

type onePrice struct{}

func (onePrice) Price(string) clicks.Price { return clicks.Price{Slowdown: 1} }

var config = clicks.ThrottleConfig{Config: cpratelimit.Config{PerSecond: 0.2, Burst: 60}, ScopeMultiplier: 10}

type fixture struct {
	refills *stubRefills
	limiter *cpratelimit.Limiter
	useCase *use_refill_usecase.UseCase
}

func setup(held bool) fixture {
	refills := &stubRefills{held: map[bonuses.Holder]bool{"a-guest": held}}
	limiter := cpratelimit.New("test", config.Config, cptime.NewFixedClock(epoch))

	return fixture{
		refills: refills,
		limiter: limiter,
		useCase: use_refill_usecase.New(refills, limiter, onePrice{}, config.Buckets()),
	}
}

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")
}

// spend clicks n times the way the throttle does, from both buckets.
func (f fixture) spend(t *testing.T, n int) {
	t.Helper()

	keys := config.Buckets().Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}, clicks.Price{Slowdown: 1})
	for range n {
		allowed, _ := f.limiter.TakeAll(1, keys...)
		require.True(t, allowed)
	}
}

func TestARefillFillsTheBankAndIsSpent(t *testing.T) {
	f := setup(true)
	f.spend(t, 50)

	out, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.InDelta(t, 60, out.Budget.Tokens, 1e-9, "the bank is full")
	assert.Equal(t, 60, out.Budget.Capacity, "and no bigger")
	assert.Equal(t, bonuses.Held{}, out.Held)
	assert.Equal(t, []bonuses.Holder{"a-guest"}, f.refills.spent)
}

func TestAFullBankIsRefusedAndSpendsNothing(t *testing.T) {
	f := setup(true)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, use_refill_usecase.ErrBankFull)
	assert.Empty(t, f.refills.spent, "a press on a full meter wastes nothing")
}

func TestNoRefillFillsNothing(t *testing.T) {
	f := setup(false)
	f.spend(t, 50)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, use_refill_usecase.ErrNoRefill)
	state := f.limiter.Peek(config.Buckets().Own(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}))
	assert.InDelta(t, 10, state.Tokens, 1e-9)
}

func TestACallerWithNoAccountHasNoRefill(t *testing.T) {
	f := setup(true)

	_, err := f.useCase.Execute(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), use_refill_usecase.In{})

	require.ErrorIs(t, err, use_refill_usecase.ErrNoRefill)
	assert.Empty(t, f.refills.spent)
}

func TestTheScopesBucketIsNotFilled(t *testing.T) {
	f := setup(true)
	f.spend(t, 50)

	_, err := f.useCase.Execute(played(t), use_refill_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	scope := f.limiter.Peek(config.Buckets().Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}, clicks.Price{})[1])
	assert.InDelta(t, 550, scope.Tokens, 1e-9, "the scope's bucket, shared by everyone behind the address, keeps its level")
}
