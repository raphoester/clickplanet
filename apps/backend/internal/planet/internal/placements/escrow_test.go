package placements_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const alice charges.Holder = "alice"

var takenAt = time.Date(2026, 11, 10, 14, 0, 0, 0, time.UTC)

type fixture struct {
	hands  *charges.MemoryStore
	kept   *placements.MemoryStore
	escrow placements.Escrow
}

func setup(t *testing.T, shields int) fixture {
	t.Helper()

	hands := charges.NewMemoryStore()
	hands.Keep(alice, charges.Held{Shields: shields})
	kept := placements.NewMemoryStore()

	return fixture{
		hands: hands,
		kept:  kept,
		escrow: placements.NewEscrow(cpatomic.FakeTransactioner{}, charges.NewCharges(hands, charges.TestRules), kept,
			&placements.SequentialIDs{}, cptime.NewFixedClock(takenAt)),
	}
}

func TestATakenShieldLeavesTheHandAndWaitsUnderWay(t *testing.T) {
	f := setup(t, 3)

	held, id, err := f.escrow.Take(t.Context(), alice)
	require.NoError(t, err)

	assert.Equal(t, charges.Held{Shields: 2}, held)
	assert.Equal(t, charges.Held{Shields: 2}, f.hands.Kept(alice))
	assert.Equal(t, []placements.Snapshot{{ID: id, Holder: alice, TakenAt: takenAt}}, f.kept.Kept())
}

func TestAShieldIsReturnedOnceHoweverOftenItIsAsked(t *testing.T) {
	f := setup(t, 3)

	_, id, err := f.escrow.Take(t.Context(), alice)
	require.NoError(t, err)
	require.NoError(t, f.escrow.Return(t.Context(), id, id))
	require.NoError(t, f.escrow.Return(t.Context(), id))

	assert.Equal(t, charges.Held{Shields: 3}, f.hands.Kept(alice))
}

func TestAShieldTheMapSavedEndsAndIsNeverReturned(t *testing.T) {
	f := setup(t, 3)

	_, id, err := f.escrow.Take(t.Context(), alice)
	require.NoError(t, err)
	require.NoError(t, f.escrow.Confirm(t.Context(), []clicks.PlacementID{id}))
	require.NoError(t, f.escrow.Return(t.Context(), id))

	assert.Equal(t, charges.Held{Shields: 2}, f.hands.Kept(alice))
	assert.Empty(t, f.kept.Kept())
}

func TestConfirmingLeavesTheOtherPlacementsUnderWay(t *testing.T) {
	f := setup(t, 3)

	_, first, err := f.escrow.Take(t.Context(), alice)
	require.NoError(t, err)
	_, second, err := f.escrow.Take(t.Context(), alice)
	require.NoError(t, err)
	require.NoError(t, f.escrow.Confirm(t.Context(), []clicks.PlacementID{first}))

	assert.Equal(t, []placements.Snapshot{{ID: second, Holder: alice, TakenAt: takenAt}}, f.kept.Kept())
}
