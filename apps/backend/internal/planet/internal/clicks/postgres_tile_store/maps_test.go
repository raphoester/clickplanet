package postgres_tile_store_test

import (
	"context"

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
	s.Require().NoError(s.store.Save(ctx, clicks.World, owned("fr", 1, 2), fortified))
	s.Require().NoError(s.store.Save(ctx, battle, append(owned("it", 2), owned("de", 3)...), nil))

	s.Equal(map[uint32]string{1: "fr", 2: "fr"}, s.loadOn(clicks.World))
	s.Equal(map[uint32]string{2: "it", 3: "de"}, s.loadOn(battle))
	locks := map[clicks.LandmassID]string{}
	s.Require().NoError(s.store.LoadLandmasses(ctx, battle, "borders.bin", func(id clicks.LandmassID, by string) { locks[id] = by }))
	s.Empty(locks)
}

func (s *testSuite) TestAWipeDeletesOneMapAndLeavesTheOthers() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, clicks.World, owned("fr", 1, 2), nil))
	s.Require().NoError(s.store.Save(ctx, battle, owned("it", 2), []inmemory_tile_storage.Landmass{{ID: 1, Asset: "borders.bin", FortifiedBy: "it"}}))

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
	s.Require().NoError(s.store.Save(ctx, clicks.World, held, nil))
	s.Require().NoError(s.store.Save(ctx, battle, held, nil))

	s.Require().NoError(s.store.ForgetHolder(ctx, ana))

	for _, played := range []clicks.MapID{clicks.World, battle} {
		s.Require().NoError(s.store.Load(ctx, played, func(tile inmemory_tile_storage.Tile) {
			s.Equal(cpsession.NoAccount, tile.Holder, "%s", played)
			s.Equal("fr", tile.Owner)
		}))
	}
}
