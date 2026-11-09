package inmemory_tile_storage_test

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// A map as a battle finds it: an island fortified by France, held by ana's take, and a German tile flushed or not.
func (s *testSuite) mapBeforeABattle(persistence *inmemory_tile_storage.MemoryPersistence) (*inmemory_tile_storage.Storage, cpsession.AccountID) {
	ctx := context.Background()
	ana := cpsession.AccountID(uuid.New())
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Click(cpctx.AddAccountToContext(ctx, ana.String()), island[0], "fr"))
	for _, tile := range island[1:] {
		s.Require().NoError(storage.Set(ctx, tile, "fr"))
	}
	_, err := storage.Fortify(ctx, island[2], "fr", 10)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))
	s.Require().NoError(storage.Set(ctx, 12, "de"))
	return storage, ana
}

func (s *testSuite) TestABlankKeepsACopyOfTheMapInMemoryWithTheTilesNotFlushedYet() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage, ana := s.mapBeforeABattle(persistence)
	var kept clicks.MapCopy

	blanked, err := storage.Blank(ctx, func(_ context.Context, copied clicks.MapCopy) (bool, error) {
		kept = copied
		return true, nil
	})

	s.Require().NoError(err)
	s.True(blanked)
	s.ElementsMatch([]clicks.TileCopy{
		{ID: island[0], Owner: "fr", Shields: 1, Holder: ana},
		{ID: island[1], Owner: "fr", Shields: 1},
		{ID: island[2], Owner: "fr", Shields: 1},
		{ID: 12, Owner: "de"},
	}, kept.Tiles)
	s.Equal([]clicks.LandmassCopy{{ID: 1, Asset: borders.Asset(), FortifiedBy: "fr"}}, kept.Landmasses)
	for _, tile := range append([]uint32{12}, island...) {
		owner, _ := storage.Owner(tile)
		s.Empty(owner, "tile %d", tile)
	}
	s.Empty(persistence.Stored())
	s.Require().NoError(storage.Flush(ctx))
	s.Empty(persistence.Stored(), "nothing of the map before it is flushed back")
}

func (s *testSuite) TestABlankKeepSaysNoToLeavesTheMap() {
	ctx := context.Background()
	storage, _ := s.mapBeforeABattle(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	_, version, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	blanked, err := storage.Blank(ctx, func(context.Context, clicks.MapCopy) (bool, error) { return false, nil })

	s.Require().NoError(err)
	s.False(blanked)
	owner, _ := storage.Owner(12)
	s.Equal("de", owner)
	_, still, err := storage.Subscribe(ctx)
	s.Require().NoError(err)
	s.Equal(version, still)
}

func (s *testSuite) TestAMapPutBackIsTheMapBeforeTileForTileInMemoryAndStored() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage, ana := s.mapBeforeABattle(persistence)
	var kept clicks.MapCopy
	_, err := storage.Blank(ctx, func(_ context.Context, copied clicks.MapCopy) (bool, error) {
		kept = copied
		return true, nil
	})
	s.Require().NoError(err)
	for _, tile := range append([]uint32{12, 13}, island...) {
		s.Require().NoError(storage.Set(ctx, tile, "it"))
	}
	_, err = storage.Fortify(ctx, island[0], "it", 3)
	s.Require().NoError(err)
	changes, before, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	put, err := storage.PutBack(ctx, func(context.Context) (clicks.MapCopy, bool, error) { return kept, true, nil })

	s.Require().NoError(err)
	s.True(put)
	for tile, owner := range map[uint32]string{12: "de", 13: "", island[0]: "fr", island[1]: "fr", island[2]: "fr"} {
		held, _ := storage.Owner(tile)
		s.Equal(owner, held, "tile %d", tile)
	}
	s.Equal(1, storage.Shields(island[1]))
	s.Equal([]clicks.Holding{{Account: ana, Flag: "fr", Tiles: 1}}, storage.HoldingsOf(1))
	s.Equal(map[clicks.LandmassID]string{1: "fr"}, storage.Fortresses())
	s.Equal(3, storage.Held("fr"))
	s.Zero(storage.Held("it"))
	s.Equal(map[uint32]string{12: "de", island[0]: "fr", island[1]: "fr", island[2]: "fr"}, persistence.Stored())
	s.Equal(map[uint32]cpsession.AccountID{island[0]: ana}, persistence.StoredHolders())
	s.Equal(map[clicks.LandmassID]string{1: "fr"}, persistence.StoredLandmasses(borders.Asset()))

	_, open := <-changes
	s.False(open, "a client opens again and reads the map back")
	_, after, err := storage.Subscribe(ctx)
	s.Require().NoError(err)
	s.Greater(after.Epoch, before.Epoch)
	s.Require().NoError(storage.Flush(ctx))
	s.Equal(map[uint32]string{12: "de", island[0]: "fr", island[1]: "fr", island[2]: "fr"}, persistence.Stored(),
		"nothing of the battle's map is flushed back")
}

func (s *testSuite) TestAPutBackThatCannotBeKeptLeavesTheMap() {
	ctx := context.Background()
	storage, _ := s.mapBeforeABattle(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	down := errors.New("postgres is down")

	put, err := storage.PutBack(ctx, func(context.Context) (clicks.MapCopy, bool, error) {
		return clicks.MapCopy{}, false, down
	})

	s.Require().ErrorIs(err, down)
	s.False(put)
	owner, _ := storage.Owner(12)
	s.Equal("de", owner)
}
