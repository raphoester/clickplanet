package postgres_account_store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	accounts.SessionsContractSuite

	db    *cppg.Postgres
	store *postgres_account_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_account_store.New(s.db)
	s.NewSessions = func() accounts.Sessions {
		s.Require().NoError(s.db.Purge(s.T().Context()))
		return s.store
	}
}

var (
	start    = time.Date(2026, 9, 15, 12, 0, 0, 123_456_000, time.UTC)
	lifetime = accounts.Lifetime{GuestTTL: time.Hour, ExtendEvery: time.Minute}
)

func (s *testSuite) TestTheTokenItselfIsNeverStored() {
	s.Require().NoError(s.store.CreateGuest(s.T().Context(),
		accounts.StartGuest(uuid.UUID{15: 1}, accounts.TokenOf("a-token"), lifetime, start)))

	var count int
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(),
		`SELECT count(*) FROM sessions WHERE token_hash = $1`, []byte("a-token")).Scan(&count))
	s.Zero(count)
}

func (s *testSuite) TestSavingMarksTheAccountSeen() {
	ctx := s.T().Context()
	guest := accounts.StartGuest(uuid.UUID{15: 1}, accounts.TokenOf("a-token"), lifetime, start)
	s.Require().NoError(s.store.CreateGuest(ctx, guest))

	later := start.Add(30 * time.Minute)
	s.Require().True(guest.ExtendIfDue(later, lifetime))
	s.Require().NoError(s.store.SaveSession(ctx, guest))

	var lastSeen time.Time
	s.Require().NoError(s.db.QueryRowContext(ctx, `SELECT last_seen_at FROM accounts WHERE id = $1`, guest.Account).Scan(&lastSeen))
	s.Equal(later, lastSeen.UTC())
}

func (s *testSuite) TestAGuestWhoseSessionFailsToInsertLeavesNoAccount() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.CreateGuest(ctx, accounts.StartGuest(uuid.UUID{15: 1}, accounts.TokenOf("a-token"), lifetime, start)))

	other := accounts.StartGuest(uuid.UUID{15: 2}, accounts.TokenOf("a-token"), lifetime, start)
	s.Require().Error(s.store.CreateGuest(ctx, other))

	var count int
	s.Require().NoError(s.db.QueryRowContext(ctx, `SELECT count(*) FROM accounts WHERE id = $1`, other.Account).Scan(&count))
	s.Zero(count)
}
