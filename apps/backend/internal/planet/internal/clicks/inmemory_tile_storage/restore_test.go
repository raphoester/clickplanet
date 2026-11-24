package inmemory_tile_storage_test

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

func (s *testSuite) TestARestoreGivesBackTheShieldsStruckOnceAndTellsTheStreams() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr", 2: "de", 3: "fr"})
	persistence.Shield(1, 1)
	persistence.Shield(2, 4)
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)
	listener, _, err := storage.Subscribe(ctx)
	s.Require().NoError(err)
	restorations := []clicks.Restoration{
		{Tile: 1, From: "fr", To: "fr", Was: 1, Shields: 3},
		{Tile: 2, From: "de", To: "fr", Shields: 2},
		{Tile: 3, From: "fr", To: "fr", Was: 1, Shields: 2},
	}

	restored, err := storage.Restore(ctx, restorations)
	s.Require().NoError(err)

	s.Equal(2, restored, "tile 3 holds no shield, not the one the ledger wrote")
	s.Equal(3, storage.Shields(1))
	owner, _ := storage.Owner(2)
	s.Equal("fr", owner)
	s.Equal(2, storage.Shields(2), "Germany's four shields went with its flag")
	s.Equal(clicks.TileUpdate{Tile: 1, Value: "fr", Previous: "fr", Shields: 3}, *(<-listener).Update)
	s.Equal(clicks.TileUpdate{Tile: 2, Value: "fr", Previous: "de", Shields: 2}, *(<-listener).Update)
	s.Equal(map[string]uint32{"fr": 5}, shieldsOf(storage.ShieldTally()))

	again, err := storage.Restore(ctx, restorations)
	s.Require().NoError(err)
	s.Zero(again)
	s.Equal(3, storage.Shields(1))

	s.Require().NoError(storage.Flush(ctx))
	s.Equal(map[uint32]int{1: 3, 2: 2}, persistence.StoredShields())
}
