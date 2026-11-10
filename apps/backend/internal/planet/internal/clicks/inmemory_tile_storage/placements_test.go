package inmemory_tile_storage_test

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func (s *testSuite) placing() (*inmemory_tile_storage.Storage, *inmemory_tile_storage.MemoryPersistence, *inmemory_tile_storage.RecordedPlacements) {
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr", 2: "fr"})
	placed := &inmemory_tile_storage.RecordedPlacements{}
	storage := inmemory_tile_storage.New(borders, inmemory_tile_storage.Config{}, persistence, cpatomic.FakeTransactioner{}, placed,
		cptime.SystemClock{}, slog.New(slog.DiscardHandler))
	s.Require().NoError(storage.Load(context.Background(), clicks.World))
	return storage, persistence, placed
}

func (s *testSuite) place(storage *inmemory_tile_storage.Storage, tile uint32, placement byte) {
	_, err := storage.Shield(context.Background(), tile, "fr", 10, clicks.PlacementID{15: placement})
	s.Require().NoError(err)
}

func (s *testSuite) TestAFlushConfirmsEachShieldItSavedOnce() {
	ctx := context.Background()
	storage, persistence, placed := s.placing()
	s.place(storage, 1, 1)
	s.place(storage, 2, 2)

	s.Require().NoError(storage.Flush(ctx))
	s.Require().NoError(storage.Flush(ctx))

	s.Equal([]clicks.PlacementID{{15: 1}, {15: 2}}, placed.Confirmed())
	s.Equal(map[uint32]int{1: 1, 2: 1}, persistence.StoredShields())
}

func (s *testSuite) TestAShieldIsNotConfirmedUntilItsTileIsSaved() {
	ctx := context.Background()
	storage, persistence, placed := s.placing()
	s.place(storage, 1, 1)
	persistence.FailWith(errors.New("connection reset"))

	s.Require().Error(storage.Flush(ctx))
	s.Empty(placed.Confirmed())

	persistence.Heal()
	s.Require().NoError(storage.Flush(ctx))
	s.Equal([]clicks.PlacementID{{15: 1}}, placed.Confirmed())
}

func (s *testSuite) TestAConfirmThatFailsKeepsTheTileForTheNextFlush() {
	ctx := context.Background()
	storage, persistence, placed := s.placing()
	s.place(storage, 1, 1)
	placed.FailWith(errors.New("connection reset"))

	s.Require().Error(storage.Flush(ctx))

	placed.FailWith(nil)
	s.Require().NoError(storage.Flush(ctx))
	s.Equal([]clicks.PlacementID{{15: 1}}, placed.Confirmed())
	s.Len(persistence.Saves(), 2, "the tile is saved again with its confirmation")
}

func (s *testSuite) TestAWipeConfirmsTheShieldsPlacedBeforeIt() {
	storage, _, placed := s.placing()
	s.place(storage, 1, 1)

	wiped, err := storage.Wipe(context.Background(), keeps(true))

	s.Require().NoError(err)
	s.True(wiped)
	s.Equal([]clicks.PlacementID{{15: 1}}, placed.Confirmed())
}

func (s *testSuite) TestAPutBackConfirmsTheShieldsPlacedOnTheArena() {
	ctx := context.Background()
	storage, _, placed := s.placing()
	blanked, err := storage.Blank(ctx, battle, keeps(true))
	s.Require().NoError(err)
	s.Require().True(blanked)
	s.Require().NoError(storage.Set(ctx, 3, "fr"))
	s.place(storage, 3, 1)

	dropped, err := storage.PutBack(ctx, battle, keeps(true))

	s.Require().NoError(err)
	s.True(dropped)
	s.Equal([]clicks.PlacementID{{15: 1}}, placed.Confirmed())
}
