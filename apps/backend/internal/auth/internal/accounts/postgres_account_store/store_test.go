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
	suite.Suite
	db    *cppg.Postgres
	store *postgres_account_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_account_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var (
	start   = time.Date(2026, 9, 15, 12, 0, 0, 123_456_000, time.UTC)
	account = uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")
	hash    = accounts.HashOf("a-token")
)

func (s *testSuite) TestAnUnknownTokenFindsNothing() {
	_, found, err := s.store.FindSession(s.T().Context(), hash)
	s.Require().NoError(err)
	s.False(found)
}

func (s *testSuite) TestACreatedGuestIsFoundByItsTokenHash() {
	s.Require().NoError(s.store.CreateGuest(s.T().Context(), account, hash, start.Add(time.Hour), start))

	session, found, err := s.store.FindSession(s.T().Context(), hash)
	s.Require().NoError(err)

	s.True(found)
	s.Equal(accounts.Session{Account: account, ExtendedAt: start, ExpiresAt: start.Add(time.Hour)}, session)
}

func (s *testSuite) TestTheTokenItselfIsNeverStored() {
	s.Require().NoError(s.store.CreateGuest(s.T().Context(), account, hash, start.Add(time.Hour), start))

	var count int
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(),
		`SELECT count(*) FROM sessions WHERE token_hash = $1`, []byte("a-token")).Scan(&count))
	s.Zero(count)
}

func (s *testSuite) TestExtendingMovesTheExpiryAndMarksTheAccountSeen() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.CreateGuest(ctx, account, hash, start.Add(time.Hour), start))

	later := start.Add(30 * time.Minute)
	s.Require().NoError(s.store.ExtendSession(ctx, hash, later.Add(time.Hour), later))

	session, _, err := s.store.FindSession(ctx, hash)
	s.Require().NoError(err)
	s.Equal(accounts.Session{Account: account, ExtendedAt: later, ExpiresAt: later.Add(time.Hour)}, session)

	var lastSeen time.Time
	s.Require().NoError(s.db.QueryRowContext(ctx, `SELECT last_seen_at FROM accounts WHERE id = $1`, account).Scan(&lastSeen))
	s.Equal(later, lastSeen.UTC())
}

func (s *testSuite) TestExtendingAnUnknownSessionFails() {
	s.Error(s.store.ExtendSession(s.T().Context(), hash, start.Add(time.Hour), start))
}

func (s *testSuite) TestAGuestWhoseSessionFailsToInsertIsNotCreated() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.CreateGuest(ctx, account, hash, start.Add(time.Hour), start))

	other := uuid.MustParse("01926c6e-0000-7000-8000-000000000001")
	s.Require().Error(s.store.CreateGuest(ctx, other, hash, start.Add(time.Hour), start), "the token hash is taken")

	var count int
	s.Require().NoError(s.db.QueryRowContext(ctx, `SELECT count(*) FROM accounts WHERE id = $1`, other).Scan(&count))
	s.Zero(count)
}
