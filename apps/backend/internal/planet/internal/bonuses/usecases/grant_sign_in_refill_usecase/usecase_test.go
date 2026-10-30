package grant_sign_in_refill_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_sign_in_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type grants struct {
	recorded *cpcolls.Set[bonuses.GrantID]
}

func (g grants) Record(_ context.Context, grant bonuses.GrantID, _ time.Time) (bool, error) {
	if g.recorded.Contains(grant) {
		return false, nil
	}
	g.recorded.Add(grant)
	return true, nil
}

const (
	ada   = bonuses.Holder("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	grace = bonuses.Holder("7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f")
)

func setUp(signInRefill bool) (*grant_sign_in_refill_usecase.UseCase, bonuses.Charges) {
	rules := bonuses.TestRules()
	rules.SignInRefill = signInRefill
	inForce := func() bonuses.Rules { return rules }
	charges := bonuses.NewCharges(bonuses.NewMemoryStore(), inForce)
	return grant_sign_in_refill_usecase.New(cpatomic.FakeTransactioner{}, grants{recorded: cpcolls.NewSet[bonuses.GrantID]()},
		charges, inForce, cptime.NewFixedClock(time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC))), charges
}

func refillHeld(t *testing.T, charges bonuses.Charges, holder bonuses.Holder) bool {
	t.Helper()

	held, err := charges.Held(t.Context(), holder)
	require.NoError(t, err)
	return held.Refill
}

func TestAFirstSignInEarnsARefill(t *testing.T) {
	useCase, charges := setUp(true)

	require.NoError(t, useCase.Execute(t.Context(), ada))

	assert.True(t, refillHeld(t, charges, ada))
}

func TestAnAccountEarnsItOnceEvenToldAgainAfterUsingIt(t *testing.T) {
	useCase, charges := setUp(true)
	require.NoError(t, useCase.Execute(t.Context(), ada))
	require.NoError(t, charges.Spend(t.Context(), ada, bonuses.Held.AfterRefill))

	require.NoError(t, useCase.Execute(t.Context(), ada))

	assert.False(t, refillHeld(t, charges, ada))
}

func TestEachAccountEarnsItsOwn(t *testing.T) {
	useCase, charges := setUp(true)

	require.NoError(t, useCase.Execute(t.Context(), ada))
	require.NoError(t, useCase.Execute(t.Context(), grace))

	assert.True(t, refillHeld(t, charges, ada))
	assert.True(t, refillHeld(t, charges, grace))
}

func TestNoRefillWhileTheRuleIsOff(t *testing.T) {
	useCase, charges := setUp(false)

	require.NoError(t, useCase.Execute(t.Context(), ada))

	assert.False(t, refillHeld(t, charges, ada))
}
