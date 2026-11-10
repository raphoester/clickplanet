package postgres_tile_store_test

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const battle = clicks.MapID("0192f5a0-7c00-7000-8000-000000000032")

func (s *testSuite) loadOn(played clicks.MapID) map[uint32]string {
	loaded := map[uint32]string{}
	s.Require().NoError(s.store.Load(context.Background(), played, func(tile inmemory_tile_storage.Tile) {
		loaded[tile.ID] = tile.Owner
	}))
	return loaded
}

func (s *testSuite) TestEachMapKeepsItsOwnTilesAndLocks() {
	ctx := context.Background()
	fortified := []inmemory_tile_storage.Landmass{{ID: 1, Asset: "borders.bin", FortifiedBy: "fr"}}
	s.Require().NoError(s.store.Save(ctx, clicks.World, owned("fr", 1, 2), fortified, nil))
	s.Require().NoError(s.store.Save(ctx, battle, append(owned("it", 2), owned("de", 3)...), nil, nil))

	s.Equal(map[uint32]string{1: "fr", 2: "fr"}, s.loadOn(clicks.World))
	s.Equal(map[uint32]string{2: "it", 3: "de"}, s.loadOn(battle))
	locks := map[clicks.LandmassID]string{}
	s.Require().NoError(s.store.LoadLandmasses(ctx, battle, "borders.bin", func(id clicks.LandmassID, by string) { locks[id] = by }))
	s.Empty(locks)
}

func (s *testSuite) TestAWipeDeletesOneMapAndLeavesTheOthers() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, clicks.World, owned("fr", 1, 2), nil, nil))
	s.Require().NoError(s.store.Save(ctx, battle, owned("it", 2), []inmemory_tile_storage.Landmass{{ID: 1, Asset: "borders.bin", FortifiedBy: "it"}}, nil))

	wiped, err := s.store.Wipe(ctx, battle, func(context.Context) (bool, error) { return true, nil })

	s.Require().NoError(err)
	s.True(wiped)
	s.Empty(s.loadOn(battle))
	s.Equal(map[uint32]string{1: "fr", 2: "fr"}, s.loadOn(clicks.World))
}

func (s *testSuite) TestAForgottenHolderHoldsNoTileOfAnyMap() {
	ctx := context.Background()
	ana := cpsession.AccountID(uuid.New())
	held := []inmemory_tile_storage.Tile{{ID: 1, Owner: "fr", Holder: ana}}
	s.Require().NoError(s.store.Save(ctx, clicks.World, held, nil, nil))
	s.Require().NoError(s.store.Save(ctx, battle, held, nil, nil))

	s.Require().NoError(s.store.ForgetHolder(ctx, ana))

	for _, played := range []clicks.MapID{clicks.World, battle} {
		s.Require().NoError(s.store.Load(ctx, played, func(tile inmemory_tile_storage.Tile) {
			s.Equal(cpsession.NoAccount, tile.Holder, "%s", played)
			s.Equal("fr", tile.Owner)
		}))
	}
}

func (s *testSuite) TestAnOpenWritesTheStartInKeepsTransaction() {
	ctx := context.Background()
	start := []inmemory_tile_storage.Tile{{ID: 4, Owner: "fr", Shields: 100}, {ID: 9, Owner: "de", Shields: 100}}

	opened, err := s.store.Open(ctx, battle, start, func(context.Context) (bool, error) { return true, nil })

	s.Require().NoError(err)
	s.True(opened)
	s.Equal(map[uint32]string{4: "fr", 9: "de"}, s.loadOn(battle))
	s.Require().NoError(s.store.Load(ctx, battle, func(tile inmemory_tile_storage.Tile) {
		s.Equal(100, tile.Shields, "tile %d", tile.ID)
		s.Equal(cpsession.NoAccount, tile.Holder, "no player's take")
	}))
	s.Empty(s.loadOn(clicks.World))
}

func (s *testSuite) TestAnOpenKeepRefusesWritesNoStart() {
	ctx := context.Background()
	start := []inmemory_tile_storage.Tile{{ID: 4, Owner: "fr", Shields: 100}}

	opened, err := s.store.Open(ctx, battle, start, func(context.Context) (bool, error) { return false, nil })
	s.Require().NoError(err)
	s.False(opened)

	down := errors.New("the arena could not be kept")
	_, err = s.store.Open(ctx, battle, start, func(context.Context) (bool, error) { return false, down })
	s.Require().ErrorIs(err, down)

	s.Empty(s.loadOn(battle))
}
