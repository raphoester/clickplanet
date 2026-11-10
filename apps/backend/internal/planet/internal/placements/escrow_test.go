package placements_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const alice bonuses.Holder = "alice"

var takenAt = time.Date(2026, 11, 10, 14, 0, 0, 0, time.UTC)

type fixture struct {
	hands  *bonuses.MemoryStore
	kept   *placements.MemoryStore
	escrow placements.Escrow
}

func setup(t *testing.T, shields int) fixture {
	t.Helper()

	hands := bonuses.NewMemoryStore()
	hands.Keep(alice, bonuses.Held{Shields: shields})
	kept := placements.NewMemoryStore()

	return fixture{
		hands: hands,
		kept:  kept,
		escrow: placements.NewEscrow(cpatomic.FakeTransactioner{}, bonuses.NewCharges(hands, bonuses.TestRules), kept,
			&placements.SequentialIDs{}, cptime.NewFixedClock(takenAt)),
	}
}

func TestATakenShieldLeavesTheHandAndWaitsUnderWay(t *testing.T) {
	f := setup(t, 3)

	held, id, err := f.escrow.Take(t.Context(), alice)
	require.NoError(t, err)

	assert.Equal(t, bonuses.Held{Shields: 2}, held)
	assert.Equal(t, bonuses.Held{Shields: 2}, f.hands.Kept(alice))
	assert.Equal(t, []placements.Snapshot{{ID: id, Holder: alice, TakenAt: takenAt}}, f.kept.Kept())
}

func TestAHandWithNoShieldTakesNone(t *testing.T) {
	f := setup(t, 0)

	_, _, err := f.escrow.Take(t.Context(), alice)

	require.ErrorIs(t, err, bonuses.ErrNotHeld)
	assert.Empty(t, f.kept.Kept())
}

func TestAShieldReturnedComesBackAboveTheStackSize(t *testing.T) {
	f := setup(t, bonuses.TestRules().Shield.Held+5)

	_, id, err := f.escrow.Take(t.Context(), alice)
	require.NoError(t, err)
	require.NoError(t, f.escrow.Return(t.Context(), id))

	assert.Equal(t, bonuses.Held{Shields: bonuses.TestRules().Shield.Held + 5}, f.hands.Kept(alice))
	assert.Empty(t, f.kept.Kept())
}

func TestAShieldIsReturnedOnceHoweverOftenItIsAsked(t *testing.T) {
	f := setup(t, 3)

	_, id, err := f.escrow.Take(t.Context(), alice)
	require.NoError(t, err)
	require.NoError(t, f.escrow.Return(t.Context(), id, id))
	require.NoError(t, f.escrow.Return(t.Context(), id))

	assert.Equal(t, bonuses.Held{Shields: 3}, f.hands.Kept(alice))
}

func TestAShieldTheMapSavedEndsAndIsNeverReturned(t *testing.T) {
	f := setup(t, 3)

	_, id, err := f.escrow.Take(t.Context(), alice)
	require.NoError(t, err)
	require.NoError(t, f.escrow.Confirm(t.Context(), []clicks.PlacementID{id}))
	require.NoError(t, f.escrow.Return(t.Context(), id))

	assert.Equal(t, bonuses.Held{Shields: 2}, f.hands.Kept(alice))
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
