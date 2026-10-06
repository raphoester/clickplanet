package postgres_tile_store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

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

	s.Require().NoError(s.store.Save(ctx, append(append(owned("fr", 0), owned("us", 7)...), owned("de", 257_948)...)))

	s.Equal(map[uint32]string{0: "fr", 7: "us", 257_948: "de"}, s.load())
}

func (s *testSuite) TestSaveOverwritesAnOwnerAndDeletesAFreedTile() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, owned("fr", 1, 2, 3)))

	s.Require().NoError(s.store.Save(ctx, append(owned("us", 1), owned("", 2, 4)...)))

	s.Equal(map[uint32]string{1: "us", 3: "fr"}, s.load(),
		"a freed tile has no row, and freeing one that had none is not an error")
}

func (s *testSuite) TestSaveSpansChunks() {
	const count = 25_000
	ids := make([]uint32, count)
	for i := range ids {
		ids[i] = uint32(i)
	}

	s.Require().NoError(s.store.Save(context.Background(), owned("bg", ids...)))

	s.Len(s.load(), count)
}

func (s *testSuite) TestAFailedSaveWritesNothing() {
	ctx := context.Background()

	ids := make([]uint32, 15_000)
	for i := range ids {
		ids[i] = uint32(i)
	}
	s.Require().NoError(s.store.Save(ctx, owned("fr", 20_000)))
	_, err := s.db.ExecContext(ctx, `ALTER TABLE tiles ADD CONSTRAINT not_bg_past_ten_thousand CHECK (id < 10000 OR country <> 'bg')`)
	s.Require().NoError(err)
	defer func() {
		_, err := s.db.ExecContext(ctx, `ALTER TABLE tiles DROP CONSTRAINT not_bg_past_ten_thousand`)
		s.Require().NoError(err)
	}()

	s.Require().Error(s.store.Save(ctx, owned("bg", ids...)))

	s.Equal(map[uint32]string{20_000: "fr"}, s.load())
}

func (s *testSuite) TestShieldsAreKeptWithTheirTile() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, []inmemory_tile_storage.Tile{
		{ID: 1, Owner: "fr", Shields: 3},
		{ID: 2, Owner: "fr"},
	}))
	s.Require().NoError(s.store.Save(ctx, []inmemory_tile_storage.Tile{{ID: 1, Owner: "fr", Shields: 2}}))

	shields := map[uint32]int{}
	s.Require().NoError(s.store.Load(ctx, func(tile uint32, _ string, held int) {
		shields[tile] = held
	}))
	s.Equal(map[uint32]int{1: 2, 2: 0}, shields)
}
