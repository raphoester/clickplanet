package postgres_worn_title_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/postgres_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	wearing.StoreContractSuite

	db    *cppg.Postgres
	store *postgres_worn_title_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.store = postgres_worn_title_store.New(s.db)
	s.NewStore = func() wearing.Store { return s.store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

func (s *testSuite) TestTheTimeOfTheLastChoiceIsKept() {
	at := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	s.Require().NoError(s.store.Wear(s.T().Context(), players.AccountID{15: 1}, "settler", at))
	s.Require().NoError(s.store.Wear(s.T().Context(), players.AccountID{15: 1}, "og", at.Add(time.Hour)))

	var wornAt time.Time
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT worn_at FROM worn_titles`).Scan(&wornAt))
	s.Equal(at.Add(time.Hour), wornAt.UTC())
}
