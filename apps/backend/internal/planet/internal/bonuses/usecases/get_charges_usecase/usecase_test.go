package get_charges_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/get_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func charges(t *testing.T) (bonuses.Charges, *bonuses.MemoryStore) {
	t.Helper()

	store := bonuses.NewMemoryStore()
	charges := bonuses.NewCharges(store, bonuses.ChargesConfig{})
	require.NoError(t, charges.Grant(t.Context(), "a-guest", bonuses.KindBomb, 1))

	return charges, store
}

func TestTheChargesReadAreTheAccounts(t *testing.T) {
	charges, _ := charges(t)
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	held, err := get_charges_usecase.New(charges).Execute(ctx)
	require.NoError(t, err)

	assert.Equal(t, bonuses.Held{Bomb: true}, held)
}

func TestWithNoAccountNothingIsHeld(t *testing.T) {
	charges, _ := charges(t)

	held, err := get_charges_usecase.New(charges).Execute(cpctx.AddIPToContext(t.Context(), "1.2.3.4"))
	require.NoError(t, err)

	assert.Equal(t, bonuses.Held{}, held)
}

func TestAStoreThatFailsFailsTheRead(t *testing.T) {
	charges, store := charges(t)
	down := errors.New("postgres is down")
	store.FailWith(down)
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	_, err := get_charges_usecase.New(charges).Execute(ctx)

	require.ErrorIs(t, err, down)
}
