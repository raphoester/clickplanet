package inmemory_tile_storage_test

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

func keeps(kept bool) func(context.Context) (bool, error) {
	return func(context.Context) (bool, error) { return kept, nil }
}

func (s *testSuite) TestAWipeLeavesABlankMapInMemoryAndStored() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	for _, tile := range island {
		s.Require().NoError(storage.Set(ctx, tile, "fr"))
	}
	s.Require().NoError(storage.Set(ctx, 12, "de"))
	_, err := storage.Fortify(ctx, 90_003, "fr", 10)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	wiped, err := storage.Wipe(ctx, keeps(true))

	s.Require().NoError(err)
	s.True(wiped)
	for _, tile := range append([]uint32{12}, island...) {
		owner, _ := storage.Owner(tile)
		s.Empty(owner, "tile %d", tile)
		s.Zero(storage.Shields(tile), "tile %d", tile)
	}
	s.Zero(storage.Held("fr"))
	s.Empty(persistence.Stored())
	s.Empty(persistence.StoredLandmasses(borders.Asset()))
	s.Require().NoError(storage.Flush(ctx))
	s.Empty(persistence.Stored(), "nothing of the map before it is flushed back")

	for _, tile := range island {
		s.Require().NoError(storage.Set(ctx, tile, "de"))
	}
	_, err = storage.Fortify(ctx, 90_001, "de", 10)
	s.Require().NoError(err, "the fortify lock went with the map")
}

func (s *testSuite) TestAWipeStartsANewEpochAndEndsEveryStream() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	storage := s.newStorage(inmemory_tile_storage.Config{})
	s.Require().NoError(storage.Set(ctx, 12, "fr"))
	changes, before, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	_, err = storage.Wipe(ctx, keeps(true))
	s.Require().NoError(err)

	_, open := <-changes
	s.False(open, "a client opens again and reads the blank map")
	_, after, err := storage.Subscribe(ctx)
	s.Require().NoError(err)
	s.Greater(after.Epoch, before.Epoch)
	s.Zero(after.Seq)
	batch, err := storage.StateBatchDense(1, 20)
	s.Require().NoError(err)
	s.Equal(after, batch.Version)
}

func (s *testSuite) TestAResetKeptAlreadyWipesNothing() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Set(ctx, 12, "fr"))
	s.Require().NoError(storage.Flush(ctx))
	_, version, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	wiped, err := storage.Wipe(ctx, keeps(false))

	s.Require().NoError(err)
	s.False(wiped)
	owner, _ := storage.Owner(12)
	s.Equal("fr", owner)
	s.Equal(map[uint32]string{12: "fr"}, persistence.Stored())
	_, still, err := storage.Subscribe(ctx)
	s.Require().NoError(err)
	s.Equal(version, still)
}

func (s *testSuite) TestAWipeThatCannotBeKeptLeavesTheMap() {
	ctx := context.Background()
	storage := s.newStorage(inmemory_tile_storage.Config{})
	s.Require().NoError(storage.Set(ctx, 12, "fr"))
	down := errors.New("postgres is down")

	wiped, err := storage.Wipe(ctx, func(context.Context) (bool, error) { return false, down })

	s.Require().ErrorIs(err, down)
	s.False(wiped)
	owner, _ := storage.Owner(12)
	s.Equal("fr", owner)
}
