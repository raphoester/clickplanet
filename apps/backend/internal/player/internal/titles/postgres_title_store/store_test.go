package postgres_title_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/postgres_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	titles.StoreContractSuite

	db    *cppg.Postgres
	store *postgres_title_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.store = postgres_title_store.New(s.db)
	s.NewStore = func() titles.Store { return s.store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

var at = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func (s *testSuite) TestATitleKeepsTheTimeItWasFirstEarned() {
	grant := titles.Holdings{{15: 1}: {"settler"}}
	s.Require().NoError(s.store.Grant(s.T().Context(), grant, at))

	s.Require().NoError(s.store.Grant(s.T().Context(), grant, at.Add(time.Hour)))

	var earnedAt time.Time
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT earned_at FROM titles`).Scan(&earnedAt))
	s.Equal(at, earnedAt.UTC())
}
