package postgres_raid_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/postgres_raid_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	raids.StorageContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	store := postgres_raid_store.New(s.db)
	s.NewStorage = func() raids.Storage { return store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}

func (s *testSuite) TestTheTableHoldsOneRowAtMost() {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(),
		`INSERT INTO raid_mode (single, started_at, ends_at) VALUES (false, now(), now())`)

	s.Require().Error(err, "one raid mode, never a second row beside it")
}
