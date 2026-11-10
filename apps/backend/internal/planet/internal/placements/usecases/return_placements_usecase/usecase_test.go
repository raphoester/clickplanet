package return_placements_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements/usecases/return_placements_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheBootGivesBackEveryShieldTheMapNeverSaved(t *testing.T) {
	hands := bonuses.NewMemoryStore()
	hands.Keep("alice", bonuses.Held{Shields: 40})
	hands.Keep("bob", bonuses.Held{Shields: 1})
	kept := placements.NewMemoryStore()
	escrow := placements.NewEscrow(cpatomic.FakeTransactioner{}, bonuses.NewCharges(hands, bonuses.TestRules), kept,
		&placements.SequentialIDs{}, cptime.NewFixedClock(time.Date(2026, 11, 10, 14, 0, 0, 0, time.UTC)))

	_, saved, err := escrow.Take(t.Context(), "alice")
	require.NoError(t, err)
	_, _, err = escrow.Take(t.Context(), "alice")
	require.NoError(t, err)
	_, _, err = escrow.Take(t.Context(), "bob")
	require.NoError(t, err)
	require.NoError(t, escrow.Confirm(t.Context(), []clicks.PlacementID{saved}))

	found, err := return_placements_usecase.New(kept, escrow).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 2, found)
	assert.Equal(t, bonuses.Held{Shields: 39}, hands.Kept("alice"), "the shield the map saved stays on the map")
	assert.Equal(t, bonuses.Held{Shields: 1}, hands.Kept("bob"))
	assert.Empty(t, kept.Kept())
}

func TestABootWithNothingLeftChangesNothing(t *testing.T) {
	kept := placements.NewMemoryStore()
	escrow := placements.NewEscrow(cpatomic.FakeTransactioner{}, bonuses.NewCharges(bonuses.NewMemoryStore(), bonuses.TestRules),
		kept, &placements.SequentialIDs{}, cptime.SystemClock{})

	found, err := return_placements_usecase.New(kept, escrow).Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, found)
}
