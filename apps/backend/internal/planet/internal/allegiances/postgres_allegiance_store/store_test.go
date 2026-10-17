package postgres_allegiance_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/postgres_allegiance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db    *cppg.Postgres
	store *postgres_allegiance_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_allegiance_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

const (
	alice allegiances.Account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	bob   allegiances.Account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"
)

var at = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

func (s *testSuite) load() map[allegiances.Account]allegiances.Snapshot {
	kept := map[allegiances.Account]allegiances.Snapshot{}
	s.Require().NoError(s.store.Load(s.T().Context(), func(account allegiances.Account, allegiance allegiances.Snapshot) {
		kept[account] = allegiance
	}))
	return kept
}

func (s *testSuite) TestAnEmptyStoreLoadsNothing() {
	s.Empty(s.load())
}

func (s *testSuite) TestSaveThenLoadEveryAllegiance() {
	kept := map[allegiances.Account]allegiances.Snapshot{
		alice: {Flag: "fr"},
		bob:   {Flag: "de", ChangedAt: at},
	}

	s.Require().NoError(s.store.Save(s.T().Context(), kept))

	loaded := s.load()
	s.Equal("fr", loaded[alice].Flag)
	s.True(loaded[alice].ChangedAt.IsZero(), "a first flag keeps no time")
	s.Equal("de", loaded[bob].Flag)
	s.True(at.Equal(loaded[bob].ChangedAt))
}

func (s *testSuite) TestASecondSaveReplacesTheAllegiance() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Save(ctx, map[allegiances.Account]allegiances.Snapshot{alice: {Flag: "fr"}}))

	s.Require().NoError(s.store.Save(ctx, map[allegiances.Account]allegiances.Snapshot{alice: {Flag: "de", ChangedAt: at}}))

	loaded := s.load()
	s.Len(loaded, 1)
	s.Equal("de", loaded[alice].Flag)
	s.True(at.Equal(loaded[alice].ChangedAt))
}
