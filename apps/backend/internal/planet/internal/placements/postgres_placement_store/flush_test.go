package postgres_placement_store_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/postgres_charge_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/postgres_tile_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements/postgres_placement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const holder bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

type confirmThenFail struct{ placements.Escrow }

func (c confirmThenFail) Confirm(ctx context.Context, placed []clicks.PlacementID) error {
	return errors.Join(c.Escrow.Confirm(ctx, placed), errors.New("the transaction dies after the confirm"))
}

type flushed struct {
	db     *cppg.Postgres
	tiles  *postgres_tile_store.Store
	store  *postgres_placement_store.Store
	escrow placements.Escrow
}

func setup(t *testing.T) flushed {
	t.Helper()

	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	charges := bonuses.NewCharges(postgres_charge_store.New(db), bonuses.TestRules)
	_, err := charges.Grant(t.Context(), holder, bonuses.KindShields, 2)
	require.NoError(t, err)
	store := postgres_placement_store.New(db)
	return flushed{
		db: db, tiles: postgres_tile_store.New(db), store: store,
		escrow: placements.NewEscrow(db, charges, store, &placements.SequentialIDs{}, cptime.SystemClock{}),
	}
}

func (f flushed) storage(t *testing.T, confirming inmemory_tile_storage.Placements) *inmemory_tile_storage.Storage {
	t.Helper()

	storage := inmemory_tile_storage.New(clicks.BordersOf(100), clicks.NoCapitals(), inmemory_tile_storage.Config{}, f.tiles, f.db, confirming,
		cptime.SystemClock{}, slog.New(slog.DiscardHandler))
	require.NoError(t, storage.Load(t.Context(), clicks.World))
	require.NoError(t, storage.Set(t.Context(), 7, "fr"))
	return storage
}

func (f flushed) storedShields(t *testing.T) int {
	t.Helper()

	shields := 0
	require.NoError(t, f.tiles.Load(t.Context(), clicks.World, func(tile inmemory_tile_storage.Tile) {
		if tile.ID == 7 {
			shields = tile.Shields
		}
	}))
	return shields
}

func (f flushed) place(t *testing.T, storage *inmemory_tile_storage.Storage) {
	t.Helper()

	_, placement, err := f.escrow.Take(t.Context(), holder)
	require.NoError(t, err)
	_, err = storage.Shield(t.Context(), 7, "fr", 10, placement)
	require.NoError(t, err)
}

func TestTheFlushThatSavesAShieldEndsItsPlacement(t *testing.T) {
	f := setup(t)
	storage := f.storage(t, f.escrow)
	f.place(t, storage)

	require.NoError(t, storage.Flush(t.Context()))

	all, err := f.store.All(t.Context())
	require.NoError(t, err)
	assert.Empty(t, all)
	assert.Equal(t, 1, f.storedShields(t))
}

func TestAFlushThatFailsKeepsNeitherTheShieldNorTheEnd(t *testing.T) {
	f := setup(t)
	storage := f.storage(t, confirmThenFail{f.escrow})
	f.place(t, storage)

	require.Error(t, storage.Flush(t.Context()))

	all, err := f.store.All(t.Context())
	require.NoError(t, err)
	assert.Len(t, all, 1, "the placement is still under way")
	assert.Zero(t, f.storedShields(t), "the shield is not saved either")
}
