package postgres_subscription_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/postgres_subscription_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	subscriptions.StoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "marketing", migrations.FS)
	store := postgres_subscription_store.New(s.db)
	s.NewStore = func() subscriptions.Store { return store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

func (s *testSuite) TestTheTableRefusesAWithdrawalWithNoTime() {
	_, err := s.db.ExecContext(s.T().Context(), `
		INSERT INTO subscriptions (account_id, address, state, consent, asked_at)
		VALUES ('0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11', 'ada@example.com', 'withdrawn', 'season-emails-1', now())
	`)

	s.Error(err)
}
