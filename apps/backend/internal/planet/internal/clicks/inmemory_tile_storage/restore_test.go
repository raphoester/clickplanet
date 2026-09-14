package inmemory_tile_storage_test

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (s *testSuite) TestRestoreGivesBackOnlyTheTilesStillHoldingFrom() {
	ctx := context.Background()
	for tile, country := range map[uint32]string{3: "ps", 4: "de", 5: "ps"} {
		s.Require().NoError(s.storage.Set(ctx, tile, country))
	}

	restored, err := s.storage.Restore(ctx, []clicks.Restoration{
		{Tile: 3, From: "ps", To: "il"},
		{Tile: 4, From: "ps", To: "il"},
		{Tile: 5, From: "ps", To: ""},
		{Tile: maxIndex + 1, From: "ps", To: "il"},
	})
	s.Require().NoError(err)

	s.Equal(2, restored)
	for tile, want := range map[uint32]string{3: "il", 4: "de", 5: ""} {
		owner, _ := s.storage.Owner(tile)
		s.Equal(want, owner, "tile %d", tile)
	}
	s.Equal(0, s.storage.Held("ps"))
	s.Equal(1, s.storage.Held("il"))
}

func (s *testSuite) TestRestorePublishesOneOrdinaryUpdatePerTile() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	s.Require().NoError(s.storage.Set(ctx, 7, "ps"))

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	_, err = s.storage.Restore(ctx, []clicks.Restoration{{Tile: 7, From: "ps", To: "il"}})
	s.Require().NoError(err)

	select {
	case <-ctx.Done():
		s.T().Fatal("timeout")
	case change := <-listener:
		s.Equal(&clicks.TileUpdate{Tile: 7, Value: "il", Previous: "ps"}, change.Update)
	}
}
