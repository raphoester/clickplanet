package postgres_round_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	rounds.StoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "seasons", migrations.FS)
	store := postgres_round_store.New(s.db)
	s.NewStore = func() rounds.Store { return store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

func (s *testSuite) TestEachSnapshotIsOneSampleAndKeepsTheSizeOfTheMap() {
	store := postgres_round_store.New(s.db)
	round := rounds.Round{EndsAt: time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC)}
	s.Require().NoError(store.RecordSnapshot(s.T().Context(), round, rounds.Snapshot{Tiles: 100, Held: map[rounds.Country]uint32{"fr": 1}}))
	s.Require().NoError(store.RecordSnapshot(s.T().Context(), round, rounds.Snapshot{Tiles: 120}))

	var samples, mapTiles int
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(),
		`SELECT samples, map_tiles FROM rounds`).Scan(&samples, &mapTiles))
	s.Equal(2, samples)
	s.Equal(120, mapTiles)
}
