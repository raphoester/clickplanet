package bonuses_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

func TestABombIsSpentOnceAndOnlyByItsHolder(t *testing.T) {
	store := bonuses.NewMemoryStore()
	charges := bonuses.NewCharges(store, bonuses.TestRules)
	require.NoError(t, charges.Grant(t.Context(), "alice", bonuses.KindBomb, 1))

	require.ErrorIs(t, charges.Spend(t.Context(), "bob", bonuses.Held.AfterBomb), bonuses.ErrNotHeld)
	require.NoError(t, charges.Spend(t.Context(), "alice", bonuses.Held.AfterBomb))
	require.ErrorIs(t, charges.Spend(t.Context(), "alice", bonuses.Held.AfterBomb), bonuses.ErrNotHeld)
	assert.Equal(t, bonuses.Held{}, store.Kept("alice"))
}

func TestACallerWithNoAccountHoldsNothing(t *testing.T) {
	charges := bonuses.NewCharges(bonuses.NewMemoryStore(), bonuses.TestRules)

	require.NoError(t, charges.Grant(t.Context(), bonuses.NoHolder, bonuses.KindBomb, 1))

	held, err := charges.Held(t.Context(), bonuses.NoHolder)
	require.NoError(t, err)
	assert.Equal(t, bonuses.Held{}, held)
	require.ErrorIs(t, charges.Spend(t.Context(), bonuses.NoHolder, bonuses.Held.AfterBomb), bonuses.ErrNotHeld)
}
