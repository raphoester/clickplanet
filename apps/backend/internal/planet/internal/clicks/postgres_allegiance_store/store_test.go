package postgres_allegiance_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/postgres_allegiance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	clicks.AllegianceStorageContractSuite
}

func (s *testSuite) SetupSuite() {
	db := cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	store := postgres_allegiance_store.New(db)

	s.NewStorage = func() clicks.AllegianceStorage {
		s.Require().NoError(db.Purge(s.T().Context()))
		return store
	}
}
