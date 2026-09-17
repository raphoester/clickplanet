package postgres_player_store_test

import (
	"strings"
	"sync"
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
	players.StoreContractSuite

	db    *cppg.Postgres
	store *postgres_player_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.store = postgres_player_store.New(s.db)
	s.NewStore = func() players.Store { return s.store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

var at = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func (s *testSuite) TestTheTableRefusesANameLongerThan24() {
	err := s.store.SaveProfile(s.T().Context(),
		players.Profile{Account: players.AccountID{15: 1}, Name: players.Name(strings.Repeat("a", 25)), UpdatedAt: at})

	s.Error(err)
}

func (s *testSuite) TestTheStreakDayIsStoredAsADate() {
	s.Require().NoError(s.store.RecordTake(s.T().Context(), players.AccountID{15: 1}, at))

	var day string
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT streak_last_day::text FROM stats`).Scan(&day))
	s.Equal("2026-09-17", day)
}

func (s *testSuite) TestConcurrentFirstTakesAreAllCounted() {
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			<-start
			s.NoError(s.store.RecordTake(s.T().Context(), players.AccountID{15: 1}, at))
		})
	}
	close(start)
	wg.Wait()

	stats, err := s.store.Stats(s.T().Context(), players.AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(uint64(20), stats.TilesTaken, "no take overwrites another, the first ones included")
}
