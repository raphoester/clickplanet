package postgres_arrival_store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/postgres_arrival_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	arrivals.StoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	store := postgres_arrival_store.New(s.db)
	s.NewStore = func() arrivals.Store { return store }
	s.Kept = func(arrivals.Store) []arrivals.Snapshot { return s.kept() }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

func (s *testSuite) kept() []arrivals.Snapshot {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(),
		`SELECT id, account_id, source, flag, arrived_at FROM arrivals ORDER BY arrived_at, id`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var kept []arrivals.Snapshot
	for rows.Next() {
		var (
			id      uuid.UUID
			account uuid.NullUUID
			source  string
			flag    string
			at      time.Time
		)
		s.Require().NoError(rows.Scan(&id, &account, &source, &flag, &at))
		kept = append(kept, arrivals.Snapshot{
			ID: arrivals.ArrivalID(id), Account: arrivals.AccountID(account.UUID), Source: arrivals.Source(source),
			Flag: flag, At: at,
		})
	}
	s.Require().NoError(rows.Err())
	return kept
}

func (s *testSuite) TestAnAnonymizedArrivalIsStoredWithNoAccount() {
	arrival, err := arrivals.Snapshot{
		ID: arrivals.ArrivalID{15: 1}, Source: arrivals.SourceYouTube, At: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC),
	}.Restore()
	s.Require().NoError(err)
	s.Require().NoError(postgres_arrival_store.New(s.db).Save(s.T().Context(), arrival))

	var account uuid.NullUUID
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(), `SELECT account_id FROM arrivals`).Scan(&account))
	s.False(account.Valid)
}

func (s *testSuite) TestASourceTheTypeDoesNotHoldIsRefused() {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(),
		`INSERT INTO arrivals (id, source, flag, arrived_at) VALUES ($1, 'tiktok', '', now())`, uuid.UUID{15: 2})

	s.Error(err)
}
