package postgres_tile_store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/postgres_tile_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db    *cppg.Postgres
	store *postgres_tile_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_tile_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) load() map[uint32]string {
	loaded := map[uint32]string{}
	s.Require().NoError(s.store.Load(context.Background(), func(tile uint32, owner string, _ int) {
		loaded[tile] = owner
	}))
	return loaded
}

func owned(owner string, ids ...uint32) []inmemory_tile_storage.Tile {
	tiles := make([]inmemory_tile_storage.Tile, len(ids))
	for i, id := range ids {
		tiles[i] = inmemory_tile_storage.Tile{ID: id, Owner: owner}
	}
	return tiles
}

func (s *testSuite) TestAnEmptyTableLoadsNothing() {
	s.Empty(s.load())
}

func (s *testSuite) TestSaveThenLoad() {
	ctx := context.Background()

	s.Require().NoError(s.store.Save(ctx, append(append(owned("fr", 0), owned("us", 7)...), owned("de", 257_948)...), nil))

	s.Equal(map[uint32]string{0: "fr", 7: "us", 257_948: "de"}, s.load())
}

func (s *testSuite) TestSaveOverwritesAnOwnerAndDeletesAFreedTile() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, owned("fr", 1, 2, 3), nil))

	s.Require().NoError(s.store.Save(ctx, append(owned("us", 1), owned("", 2, 4)...), nil))

	s.Equal(map[uint32]string{1: "us", 3: "fr"}, s.load(),
		"a freed tile has no row, and freeing one that had none is not an error")
}

func (s *testSuite) TestSaveSpansChunks() {
	const count = 25_000
	ids := make([]uint32, count)
	for i := range ids {
		ids[i] = uint32(i)
	}

	s.Require().NoError(s.store.Save(context.Background(), owned("bg", ids...), nil))

	s.Len(s.load(), count)
}

func (s *testSuite) TestAFailedSaveWritesNothing() {
	ctx := context.Background()

	ids := make([]uint32, 15_000)
	for i := range ids {
		ids[i] = uint32(i)
	}
	s.Require().NoError(s.store.Save(ctx, owned("fr", 20_000), nil))
	_, err := s.db.ExecContext(ctx, `ALTER TABLE tiles ADD CONSTRAINT not_bg_past_ten_thousand CHECK (id < 10000 OR country <> 'bg')`)
	s.Require().NoError(err)
	defer func() {
		_, err := s.db.ExecContext(ctx, `ALTER TABLE tiles DROP CONSTRAINT not_bg_past_ten_thousand`)
		s.Require().NoError(err)
	}()

	s.Require().Error(s.store.Save(ctx, owned("bg", ids...), nil))

	s.Equal(map[uint32]string{20_000: "fr"}, s.load())
}

func (s *testSuite) TestShieldsAreKeptWithTheirTile() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, []inmemory_tile_storage.Tile{
		{ID: 1, Owner: "fr", Shields: 3},
		{ID: 2, Owner: "fr"},
	}, nil))
	s.Require().NoError(s.store.Save(ctx, []inmemory_tile_storage.Tile{{ID: 1, Owner: "fr", Shields: 2}}, nil))

	shields := map[uint32]int{}
	s.Require().NoError(s.store.Load(ctx, func(tile uint32, _ string, held int) {
		shields[tile] = held
	}))
	s.Equal(map[uint32]int{1: 2, 2: 0}, shields)
}

func (s *testSuite) TestALandmassLockIsKeptForItsMapAlone() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, nil, []inmemory_tile_storage.Landmass{
		{ID: 3, Asset: "borders-a.bin", FortifiedBy: "fr"},
		{ID: 4, Asset: "borders-a.bin", FortifiedBy: "de"},
		{ID: 3, Asset: "borders-b.bin", FortifiedBy: "us"},
	}))
	s.Require().NoError(s.store.Save(ctx, nil, []inmemory_tile_storage.Landmass{
		{ID: 3, Asset: "borders-a.bin", FortifiedBy: "bg"},
	}))

	s.Equal(map[clicks.LandmassID]string{3: "bg", 4: "de"}, s.landmasses("borders-a.bin"))
	s.Equal(map[clicks.LandmassID]string{3: "us"}, s.landmasses("borders-b.bin"))
	s.Empty(s.landmasses("borders-c.bin"))
}

func (s *testSuite) landmasses(asset string) map[clicks.LandmassID]string {
	held := map[clicks.LandmassID]string{}
	s.Require().NoError(s.store.LoadLandmasses(context.Background(), asset,
		func(landmass clicks.LandmassID, fortifiedBy string) { held[landmass] = fortifiedBy }))
	return held
}
