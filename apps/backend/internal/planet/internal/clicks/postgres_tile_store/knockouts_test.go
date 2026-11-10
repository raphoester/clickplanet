package postgres_tile_store_test

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

func (s *testSuite) outOn(played clicks.MapID) []inmemory_tile_storage.OutFlag {
	var out []inmemory_tile_storage.OutFlag
	s.Require().NoError(s.store.LoadOutFlags(context.Background(), played, func(flag inmemory_tile_storage.OutFlag) {
		out = append(out, flag)
	}))
	return out
}

func (s *testSuite) TestAFlagOutIsKeptWithItsArenasTilesOnce() {
	ctx := context.Background()
	at := time.Date(2026, 11, 11, 20, 3, 4, 500_000_000, time.UTC)
	out := []inmemory_tile_storage.OutFlag{{Flag: "fr", By: "de", At: at}}

	s.Require().NoError(s.store.Save(ctx, battle, owned("de", 4), nil, out))
	s.Require().NoError(s.store.Save(ctx, battle, nil, nil, []inmemory_tile_storage.OutFlag{{Flag: "fr", By: "it", At: at.Add(time.Minute)}}),
		"a flush told it again changes nothing")

	s.Equal(out, s.outOn(battle))
	s.Empty(s.outOn(clicks.World))
}

func (s *testSuite) TestAWipeDeletesTheFlagsOutOfItsMap() {
	ctx := context.Background()
	other := clicks.MapID("0192f5a0-7c00-7000-8000-000000000033")
	s.Require().NoError(s.store.Save(ctx, battle, nil, nil, []inmemory_tile_storage.OutFlag{{Flag: "fr", By: "de", At: time.Now()}}))
	s.Require().NoError(s.store.Save(ctx, other, nil, nil, []inmemory_tile_storage.OutFlag{{Flag: "pl", By: "de", At: time.Now()}}))

	_, err := s.store.Wipe(ctx, battle, func(context.Context) (bool, error) { return true, nil })

	s.Require().NoError(err)
	s.Empty(s.outOn(battle))
	s.Len(s.outOn(other), 1)
}

func (s *testSuite) TestTheWorldsMapKeepsNoFlagOut() {
	s.Require().Error(s.store.Save(context.Background(), clicks.World, nil, nil,
		[]inmemory_tile_storage.OutFlag{{Flag: "fr", By: "de", At: time.Now()}}))
}
