package postgres_stats_store_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/postgres_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	stats.StoreContractSuite

	db    *cppg.Postgres
	store *postgres_stats_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "progression", migrations.FS)
	s.store = postgres_stats_store.New(s.db)
	s.NewStore = func() stats.Store { return s.store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

var at = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func (s *testSuite) TestTheStreakDayIsStoredAsADate() {
	s.Require().NoError(s.store.Record(s.T().Context(), accounts.AccountID{15: 1}, stats.TakeAt(0, 0, at)))

	var day string
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(), `SELECT streak_last_day::text FROM stats`).Scan(&day))
	s.Equal("2026-09-17", day)
}

func (s *testSuite) TestFactsEachSubscriberCountsAtOnceAreAllKept() {
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, change := range []func(uint64) func(stats.Stats) stats.Stats{
		func(i uint64) func(stats.Stats) stats.Stats { return stats.TakeAt(i, 0, at) },
		stats.BombAt,
		func(i uint64) func(stats.Stats) stats.Stats { return stats.FortifyAt(i, 1) },
		func(i uint64) func(stats.Stats) stats.Stats {
			return stats.MessageAt(at.Add(time.Duration(i) * time.Second)) //nolint:gosec // i < 20.
		},
	} {
		wg.Go(func() {
			<-start
			for i := range uint64(20) {
				s.NoError(s.store.Record(s.T().Context(), accounts.AccountID{15: 1}, change(i)))
			}
		})
	}
	close(start)
	wg.Wait()

	counted, err := s.store.Stats(s.T().Context(), accounts.AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(uint64(20), counted.TilesTaken(), "no change overwrites another, the first ones included")
	s.Equal(uint64(20), counted.BombsDropped())
	s.Equal(uint64(20), counted.TilesFortified())
	s.Equal(uint64(20), counted.MessagesSent())
}
