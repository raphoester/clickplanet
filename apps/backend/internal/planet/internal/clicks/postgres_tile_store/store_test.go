package postgres_tile_store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

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

func (s *testSuite) SetupTest() {
	s.db = cppg.ForTests(s.T(), "planet", migrations.FS)
	s.store = postgres_tile_store.New(s.db)
}

func (s *testSuite) load() map[uint32]string {
	loaded := map[uint32]string{}
	s.Require().NoError(s.store.Load(context.Background(), func(tile uint32, owner string) {
		loaded[tile] = owner
	}))
	return loaded
}

func (s *testSuite) TestAnEmptyTableLoadsNothing() {
	s.Empty(s.load())
}

func (s *testSuite) TestSaveThenLoad() {
	ctx := context.Background()

	s.Require().NoError(s.store.Save(ctx, []uint32{0, 7, 257_948}, []string{"fr", "us", "de"}))

	s.Equal(map[uint32]string{0: "fr", 7: "us", 257_948: "de"}, s.load())
}

func (s *testSuite) TestSaveOverwritesAnOwnerAndDeletesAFreedTile() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, []uint32{1, 2, 3}, []string{"fr", "fr", "fr"}))

	s.Require().NoError(s.store.Save(ctx, []uint32{1, 2, 4}, []string{"us", "", ""}))

	s.Equal(map[uint32]string{1: "us", 3: "fr"}, s.load(),
		"a freed tile has no row, and freeing one that had none is not an error")
}

func (s *testSuite) TestSaveSpansChunks() {
	const count = 25_000
	tiles := make([]uint32, count)
	owners := make([]string, count)
	for i := range tiles {
		tiles[i] = uint32(i)
		owners[i] = "bg"
	}

	s.Require().NoError(s.store.Save(context.Background(), tiles, owners))

	s.Len(s.load(), count)
}

func (s *testSuite) TestAFailedSaveWritesNothing() {
	ctx := context.Background()

	// The second chunk breaks the CHECK on country; the first must not have landed either.
	tiles := make([]uint32, 15_000)
	owners := make([]string, 15_000)
	for i := range tiles {
		tiles[i] = uint32(i)
		owners[i] = "bg"
	}
	s.Require().NoError(s.store.Save(ctx, []uint32{20_000}, []string{"fr"}))
	_, err := s.db.ExecContext(ctx, `ALTER TABLE tiles ADD CONSTRAINT not_bg_past_ten_thousand CHECK (id < 10000 OR country <> 'bg')`)
	s.Require().NoError(err)
	defer func() {
		_, err := s.db.ExecContext(ctx, `ALTER TABLE tiles DROP CONSTRAINT not_bg_past_ten_thousand`)
		s.Require().NoError(err)
	}()

	s.Require().Error(s.store.Save(ctx, tiles, owners))

	s.Equal(map[uint32]string{20_000: "fr"}, s.load())
}

func (s *testSuite) TestMismatchedLengthsAreRefused() {
	s.Require().Error(s.store.Save(context.Background(), []uint32{1, 2}, []string{"fr"}))
}
