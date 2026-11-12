package postgres_departure_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures/postgres_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	departures.StoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "progression", migrations.FS)
	store := postgres_departure_store.New(s.db)
	s.NewStore = func() departures.Store { return store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}
