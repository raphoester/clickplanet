package postgres_garrison_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons/postgres_garrison_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db    *cppg.Postgres
	store *postgres_garrison_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_garrison_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) load() []garrisons.Garrison {
	var loaded []garrisons.Garrison
	s.Require().NoError(s.store.Load(s.T().Context(), func(garrison garrisons.Garrison) {
		loaded = append(loaded, garrison)
	}))
	return loaded
}

func (s *testSuite) TestAnEmptyStoreLoadsNothing() {
	s.Empty(s.load())
}

func (s *testSuite) TestSaveThenLoadEveryGarrison() {
	saved := []garrisons.Garrison{
		{Tile: 7, Country: "fr", Defenders: 3},
		{Tile: 262119, Country: "de", Defenders: 10},
	}

	s.Require().NoError(s.store.Save(s.T().Context(), saved))

	s.ElementsMatch(saved, s.load())
}

func (s *testSuite) TestASecondSaveReplacesTheGarrison() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Save(ctx, []garrisons.Garrison{{Tile: 7, Country: "fr", Defenders: 3}}))

	s.Require().NoError(s.store.Save(ctx, []garrisons.Garrison{{Tile: 7, Country: "de", Defenders: 1}}))

	s.Equal([]garrisons.Garrison{{Tile: 7, Country: "de", Defenders: 1}}, s.load())
}

func (s *testSuite) TestAGarrisonWithNoDefenderLeftIsDeleted() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Save(ctx, []garrisons.Garrison{
		{Tile: 7, Country: "fr", Defenders: 1},
		{Tile: 8, Country: "fr", Defenders: 2},
	}))

	s.Require().NoError(s.store.Save(ctx, []garrisons.Garrison{{Tile: 7}}))

	s.Equal([]garrisons.Garrison{{Tile: 8, Country: "fr", Defenders: 2}}, s.load())
}
