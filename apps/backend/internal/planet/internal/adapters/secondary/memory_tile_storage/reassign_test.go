package memory_tile_storage_test

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/memory_tile_storage"
)

func (s *testSuite) TestReassignMovesOnlyTheTilesFromHolds() {
	ctx := context.Background()
	for tile, country := range map[uint32]string{3: "dz", 4: "bg", 5: "dz", 9: "fr"} {
		s.Require().NoError(s.storage.Set(ctx, tile, country))
	}

	next, moved, err := s.storage.Reassign(ctx, "dz", "fr", 0, 100)
	s.Require().NoError(err)

	s.Equal(uint32(0), next, "the scan reached the end of the map")
	s.Equal(2, moved)
	for tile, want := range map[uint32]string{3: "fr", 4: "bg", 5: "fr", 9: "fr"} {
		owner, _ := s.storage.Owner(tile)
		s.Equal(want, owner, "tile %d", tile)
	}
	s.Equal(0, s.storage.Held("dz"))
	s.Equal(3, s.storage.Held("fr"))
	s.InDelta(3.0/maxIndex, s.storage.Share("fr"), 1e-12, "the toll reads the same counts")
}

func (s *testSuite) TestReassignPublishesOneOrdinaryUpdatePerTile() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	s.Require().NoError(s.storage.Set(ctx, 7, "dz"))
	s.Require().NoError(s.storage.Set(ctx, 8, "dz"))

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	_, _, err = s.storage.Reassign(ctx, "dz", "fr", 0, 100)
	s.Require().NoError(err)

	for _, tile := range []uint32{7, 8} {
		select {
		case <-ctx.Done():
			s.T().Fatal("timeout")
		case change := <-listener:
			s.Require().NotNil(change.Update)
			s.Equal(tile, change.Update.Tile)
			s.Equal("fr", change.Update.Value)
			s.Equal("dz", change.Update.Previous, "the client moves the tile off the right score")
		}
	}
	s.Empty(listener)
}

func (s *testSuite) TestReassignStopsAtTheLimitAndResumesWhereItStopped() {
	ctx := context.Background()
	for _, tile := range []uint32{10, 20, 30, 40, 50} {
		s.Require().NoError(s.storage.Set(ctx, tile, "dz"))
	}

	next, moved, err := s.storage.Reassign(ctx, "dz", "fr", 0, 2)
	s.Require().NoError(err)
	s.Equal(2, moved)
	s.Equal(uint32(30), next, "the next tile still held, not the one after the last moved")

	next, moved, err = s.storage.Reassign(ctx, "dz", "fr", next, 2)
	s.Require().NoError(err)
	s.Equal(2, moved)
	s.Equal(uint32(50), next)

	next, moved, err = s.storage.Reassign(ctx, "dz", "fr", next, 2)
	s.Require().NoError(err)
	s.Equal(1, moved)
	s.Equal(uint32(0), next)

	s.Equal(0, s.storage.Held("dz"))
}

func (s *testSuite) TestReassignOfACountryWithNoTilesDoesNothing() {
	ctx := context.Background()
	s.Require().NoError(s.storage.Set(ctx, 1, "fr"))

	next, moved, err := s.storage.Reassign(ctx, "dz", "fr", 0, 10)
	s.Require().NoError(err)
	s.Equal(uint32(0), next)
	s.Zero(moved)

	next, moved, err = s.storage.Reassign(ctx, "fr", "fr", 0, 10)
	s.Require().NoError(err)
	s.Equal(uint32(0), next)
	s.Zero(moved, "a country is not reassigned to itself")
}

func (s *testSuite) TestReassignRefusesANonPositiveLimit() {
	_, _, err := s.storage.Reassign(context.Background(), "dz", "fr", 0, 0)
	s.Error(err)
}

func (s *testSuite) TestReassignedTilesSurviveAFlush() {
	ctx := context.Background()
	persistence := newFakePersistence()

	storage := s.newStorageOn(memory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Set(ctx, 12, "dz"))
	_, _, err := storage.Reassign(ctx, "dz", "fr", 0, 10)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	restored := s.newStorageOn(memory_tile_storage.Config{}, persistence)
	s.Require().NoError(restored.Load(ctx))
	owner, _ := restored.Owner(12)
	s.Equal("fr", owner)
	s.Equal(1, restored.Held("fr"))
	s.Zero(restored.Held("dz"))
}
