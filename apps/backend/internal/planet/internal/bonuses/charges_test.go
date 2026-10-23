package bonuses_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

const (
	alice bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	bob   bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"
)

var rules = bonuses.ChargesConfig{SpreadClicks: 8, Enclosures: 3, EnclosureMaxTiles: 25, Shields: 12}

func charges() (bonuses.Charges, *bonuses.MemoryStore) {
	store := bonuses.NewMemoryStore()
	return bonuses.NewCharges(store, rules), store
}

func TestAChargeBelongsToItsHolder(t *testing.T) {
	charges, store := charges()
	require.NoError(t, charges.Grant(t.Context(), alice, bonuses.KindBomb, 1))

	require.ErrorIs(t, charges.SpendBomb(t.Context(), bob), bonuses.ErrNotHeld)
	require.NoError(t, charges.SpendBomb(t.Context(), alice))
	require.ErrorIs(t, charges.SpendBomb(t.Context(), alice), bonuses.ErrNotHeld, "a bomb that went off is gone")
	assert.Equal(t, bonuses.Held{}, store.Kept(alice))
}

func TestEachSpendTakesItsOwnKind(t *testing.T) {
	charges, store := charges()
	for _, kind := range bonuses.Kinds {
		require.NoError(t, charges.Grant(t.Context(), alice, kind, 2))
	}

	require.NoError(t, charges.SpendRefill(t.Context(), alice))
	require.NoError(t, charges.SpendEnclose(t.Context(), alice))
	require.NoError(t, charges.SpendSpreadClick(t.Context(), alice))
	require.NoError(t, charges.SpendShield(t.Context(), alice))

	assert.Equal(t, bonuses.Held{Bomb: true, Enclosures: 1, SpreadClicks: 1, Shields: 1}, store.Kept(alice))
}

func TestAGrantStopsAtWhatFits(t *testing.T) {
	charges, store := charges()

	require.NoError(t, charges.Grant(t.Context(), alice, bonuses.KindSpreadClicks, 100))

	assert.Equal(t, bonuses.Held{SpreadClicks: 8}, store.Kept(alice))
}

func TestACallerWithNoAccountHoldsNothingAndSpendsNothing(t *testing.T) {
	charges, store := charges()
	store.FailWith(errors.New("no account is ever asked about"))

	require.NoError(t, charges.Grant(t.Context(), bonuses.NoHolder, bonuses.KindBomb, 1))
	held, err := charges.Held(t.Context(), bonuses.NoHolder)
	require.NoError(t, err)
	assert.Equal(t, bonuses.Held{}, held)
	require.ErrorIs(t, charges.SpendBomb(t.Context(), bonuses.NoHolder), bonuses.ErrNotHeld)
}

func TestAStoreThatFailsFailsTheCharge(t *testing.T) {
	charges, store := charges()
	down := errors.New("postgres is down")
	store.FailWith(down)

	require.ErrorIs(t, charges.Grant(t.Context(), alice, bonuses.KindBomb, 1), down)
	require.ErrorIs(t, charges.SpendBomb(t.Context(), alice), down)
	_, err := charges.Held(t.Context(), alice)
	require.ErrorIs(t, err, down)
}
