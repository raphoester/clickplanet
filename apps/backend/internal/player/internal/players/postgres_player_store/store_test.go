package postgres_player_store_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	players.PersistenceContractSuite

	db    *cppg.Postgres
	store *postgres_player_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.store = postgres_player_store.New(s.db)
	s.NewPersistence = func() players.Persistence { return s.store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.PersistenceContractSuite.SetupTest()
}

var at = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func (s *testSuite) TestAFailedSaveWritesNothing() {
	ctx := s.T().Context()
	stats := players.Stats{Account: players.AccountID{15: 1}, TilesTaken: 1, StreakCurrent: 1, StreakBest: 1, StreakLastDay: players.DayOf(at)}

	err := s.store.Save(ctx, players.Changes{
		Stats:    []players.Stats{stats},
		Profiles: []players.Profile{{Account: players.AccountID{15: 1}, Name: players.Name(strings.Repeat("a", 25)), UpdatedAt: at}},
	})

	s.Require().Error(err, "the table refuses a name longer than 24")
	snapshot, err := s.store.Load(ctx)
	s.Require().NoError(err)
	s.Empty(snapshot.Stats, "the stats written in the same transaction are rolled back")
}

func (s *testSuite) TestTheStreakDayIsStoredAsADate() {
	ctx := s.T().Context()
	stats := players.Stats{Account: players.AccountID{15: 1}, TilesTaken: 1, StreakCurrent: 1, StreakBest: 1, StreakLastDay: players.DayOf(at)}
	s.Require().NoError(s.store.Save(ctx, players.Changes{Stats: []players.Stats{stats}}))

	var day string
	s.Require().NoError(s.db.QueryRowContext(ctx, `SELECT streak_last_day::text FROM stats`).Scan(&day))
	s.Equal("2026-09-17", day)
}
