package inmemory_tile_storage_test

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

func (s *testSuite) TestLoadRestoresTheMapAndTheShares() {
	rows := map[uint32]string{1: "bg", 2: "bg", 3: "fr", maxIndex: "de"}
	persistence := inmemory_tile_storage.NewMemoryPersistence(rows)

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(context.Background()))

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Equal(rows, state)
	s.InDelta(2.0/maxIndex, storage.Share("bg"), 1e-12)
	s.InDelta(1.0/maxIndex, storage.Share("fr"), 1e-12)
}

func (s *testSuite) TestLoadSkipsTilesPastTheEndOfTheMap() {
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{10: "fr", maxIndex + 1: "us"})

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(context.Background()))

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Equal(map[uint32]string{10: "fr"}, state)
}

func (s *testSuite) TestAFailedLoadRefusesTheBoot() {
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	persistence.FailWith(errors.New("connection refused"))

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().ErrorContains(storage.Load(context.Background()), "connection refused")
}

func (s *testSuite) TestFlushWritesEachChangedTileOnceAsItIsNow() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Set(ctx, 1, "us"))
	s.Require().NoError(storage.Set(ctx, 70, "fr"))
	s.Require().NoError(storage.Set(ctx, 2, "fr"))
	_, err := storage.Clear(ctx, clicks.Blast{Cleared: []uint32{2}})
	s.Require().NoError(err)

	s.Require().NoError(storage.Flush(ctx))

	s.Equal([][]inmemory_tile_storage.Tile{{{ID: 1, Owner: "us"}, {ID: 2}, {ID: 70, Owner: "fr"}}}, persistence.Saves())
	s.Equal(map[uint32]string{1: "us", 70: "fr"}, persistence.Stored())
}

func (s *testSuite) TestShieldsAreKeptWithTheirTileAndGoneWithItsFlag() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Set(ctx, 2, "fr"))
	s.Require().NoError(storage.Shield(ctx, 1, "fr", 10))
	s.Require().NoError(storage.Shield(ctx, 1, "fr", 10))
	s.Require().NoError(storage.Shield(ctx, 2, "fr", 10))
	s.Require().NoError(storage.Set(ctx, 2, "de"))
	s.Require().NoError(storage.Flush(ctx))

	s.Equal(map[uint32]int{1: 2}, persistence.StoredShields(), "a tile that changed hands lost its shields")

	restarted := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(restarted.Load(ctx))
	s.Equal(2, restarted.Shields(1))
	s.Zero(restarted.Shields(2))
}

func (s *testSuite) TestFlushWithNothingChangedWritesNothing() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Flush(ctx))
	s.Require().NoError(storage.Flush(ctx))
	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Flush(ctx))

	s.Len(persistence.Saves(), 1, "a click on a tile already held changes nothing to write")
}

func (s *testSuite) TestAFailedFlushKeepsTheTilesForTheNextOne() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	persistence.FailWith(errors.New("connection reset"))
	s.Require().Error(storage.Flush(ctx))

	persistence.Heal()
	s.Require().NoError(storage.Set(ctx, 2, "us"))
	s.Require().NoError(storage.Flush(ctx))

	s.Equal(map[uint32]string{1: "fr", 2: "us"}, persistence.Stored())
}

func (s *testSuite) TestRestoredTilesAreFlushed() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 3, "ps"))
	s.Require().NoError(storage.Flush(ctx))
	_, err := storage.Restore(ctx, []clicks.Restoration{{Tile: 3, From: "ps", To: ""}})
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	s.Empty(persistence.Stored())
}

func (s *testSuite) TestRunFlushesOnShutdown() {
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{FlushInterval: time.Hour}, persistence)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		storage.Run(ctx)
	}()

	s.Require().NoError(storage.Set(context.Background(), 7, "fr"))
	cancel()

	s.Require().Eventually(func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond, "Run did not return after the context was cancelled")

	s.Equal(map[uint32]string{7: "fr"}, persistence.Stored())
}

func (s *testSuite) TestRunFlushesOnItsInterval() {
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{FlushInterval: 10 * time.Millisecond}, persistence)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go storage.Run(ctx)

	s.Require().NoError(storage.Set(context.Background(), 7, "fr"))

	s.Eventually(func() bool {
		return persistence.Stored()[7] == "fr"
	}, 5*time.Second, 10*time.Millisecond)
}

func (s *testSuite) TestReassignedTilesSurviveAFlush() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Set(ctx, 12, "dz"))
	_, _, err := storage.Reassign(ctx, "dz", "fr", 0, 10)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	restored := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(restored.Load(ctx))
	owner, _ := restored.Owner(12)
	s.Equal("fr", owner)
	s.Equal(1, restored.Held("fr"))
	s.Zero(restored.Held("dz"))
}

func (s *testSuite) TestAFortifiedLandmassStaysLockedAcrossARestart() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	for _, tile := range island {
		s.Require().NoError(storage.Set(ctx, tile, "fr"))
	}
	_, err := storage.Fortify(ctx, 90_003, "fr", 10)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	restarted := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(restarted.Load(ctx))
	s.Require().NoError(restarted.Set(ctx, 90_001, "de"))
	s.Require().NoError(restarted.Set(ctx, 90_001, "fr"))
	_, err = restarted.Fortify(ctx, 90_001, "fr", 10)

	s.Require().ErrorIs(err, clicks.ErrFortifiedAlready)
	s.Equal(map[clicks.LandmassID]string{1: "fr"}, persistence.StoredLandmasses(borders.Asset()))
	s.Equal(1, restarted.Shields(90_002))
}

func (s *testSuite) TestAWholeLandmassNobodyFortifiedIsLockedToItsHolderAtLoad() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{90_001: "fr", 90_002: "fr", 90_003: "fr"})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(ctx))
	s.Require().NoError(storage.Flush(ctx))

	_, err := storage.Fortify(ctx, 90_001, "fr", 10)

	s.Require().ErrorIs(err, clicks.ErrFortifiedAlready)
	s.Equal(map[clicks.LandmassID]string{1: "fr"}, persistence.StoredLandmasses(borders.Asset()))
	s.Zero(storage.Shields(90_001))
}

func (s *testSuite) TestALockKeptForAnotherMapIsIgnored() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{90_001: "de", 90_002: "fr", 90_003: "fr"})
	persistence.Fortified("borders-older.bin", 1, "fr")
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(ctx))
	s.Require().NoError(storage.Set(ctx, 90_001, "fr"))

	_, err := storage.Fortify(ctx, 90_001, "fr", 10)

	s.Require().NoError(err)
}
