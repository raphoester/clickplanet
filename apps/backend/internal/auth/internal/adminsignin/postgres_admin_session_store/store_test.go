package postgres_admin_session_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin/postgres_admin_session_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db       *cppg.Postgres
	accounts *postgres_account_store.Store
	store    *postgres_admin_session_store.Store
}

var (
	start    = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	ada      = accounts.AccountID{15: 1}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.accounts = postgres_account_store.New(s.db)
	s.store = postgres_admin_session_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) signedIn(account accounts.AccountID, provider, subject string) {
	s.Require().NoError(s.accounts.CreateGuest(s.T().Context(),
		accounts.GuestSession(account, accounts.TokenOf("guest-"+subject), lifetime, start)))
	identity := accounts.NewIdentity(provider, accounts.ClaimOf(subject, "", false), account, start)
	session := accounts.LinkedSession(account, accounts.TokenOf("linked-"+subject), lifetime, start)
	s.Require().NoError(s.accounts.SaveSignIn(s.T().Context(), accounts.NewSignIn(session).WithIdentity(identity)))
}

func (s *testSuite) operates(account accounts.AccountID) {
	_, err := s.db.ExecContext(s.T().Context(), `UPDATE accounts SET operator = true WHERE id = $1`, account.String())
	s.Require().NoError(err)
}

func (s *testSuite) TestAGoogleIdentityIsItsAccountAndWhetherItOperates() {
	s.signedIn(ada, "google", "ada-sub")

	candidate, err := s.store.Candidate(s.T().Context(), "google", "ada-sub")
	s.Require().NoError(err)
	s.Equal(adminsignin.CandidateOf(ada, false), candidate)

	s.operates(ada)
	candidate, err = s.store.Candidate(s.T().Context(), "google", "ada-sub")
	s.Require().NoError(err)
	s.Equal(adminsignin.CandidateOf(ada, true), candidate)
}

func (s *testSuite) TestAnIdentityNoAccountHoldsIsUnknown() {
	s.signedIn(ada, "discord", "ada-sub")
	s.operates(ada)

	for _, subject := range []string{"ada-sub", "somebody-else"} {
		_, err := s.store.Candidate(s.T().Context(), "google", subject)
		s.Require().ErrorIs(err, adminsignin.ErrUnknownIdentity, subject)
	}
}

func (s *testSuite) TestASessionIsKeptUntilItIsDeletedOrItsAccountIs() {
	s.signedIn(ada, "google", "ada-sub")
	keep := func(token string) {
		s.Require().NoError(s.store.SaveSession(s.T().Context(), adminsignin.Snapshot{
			TokenHash: accounts.TokenOf(token).Hash(), Account: ada, CreatedAt: start, ExpiresAt: start.Add(time.Hour),
		}))
	}
	count := func() int {
		var n int
		s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT count(*) FROM admin_sessions`).Scan(&n))
		return n
	}

	keep("laptop")
	keep("phone")
	s.Equal(2, count())

	s.Require().NoError(s.store.DeleteSession(s.T().Context(), accounts.TokenOf("laptop").Hash()))
	s.Equal(1, count())

	s.Require().NoError(s.accounts.DeleteAccount(s.T().Context(), ada))
	s.Equal(0, count(), "deleted with its account")
}
