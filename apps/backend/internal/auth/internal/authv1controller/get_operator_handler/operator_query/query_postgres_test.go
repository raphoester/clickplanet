package operator_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin/postgres_admin_session_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_operator_handler/operator_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db       *cppg.Postgres
	accounts *postgres_account_store.Store
	sessions *postgres_admin_session_store.Store
}

var (
	start    = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	ada      = accounts.AccountID{15: 1}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.accounts = postgres_account_store.New(s.db)
	s.sessions = postgres_admin_session_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) player(account accounts.AccountID, gameToken string) {
	session := accounts.GuestSession(account, accounts.TokenOf(gameToken), lifetime, start)
	s.Require().NoError(s.accounts.CreateGuest(s.T().Context(), session))
}

func (s *testSuite) operates(account accounts.AccountID, operates bool) {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `UPDATE accounts SET operator = $2 WHERE id = $1`, account.String(), operates)
	s.Require().NoError(err)
}

func (s *testSuite) adminSession(account accounts.AccountID, token string) time.Time {
	expiresAt := start.Add(adminsignin.DefaultTTL)
	s.Require().NoError(s.sessions.SaveSession(s.T().Context(), adminsignin.Snapshot{
		TokenHash: accounts.TokenOf(token).Hash(), Account: account, CreatedAt: start, ExpiresAt: expiresAt,
	}))
	return expiresAt
}

func (s *testSuite) operator(cookieHeader string, at time.Time) (string, error) {
	answer, err := operator_query.NewPostgresQuery(s.db, cptime.NewFixedClock(at)).Operator(s.T().Context(), cookieHeader)
	return answer.GetAccountId(), err //nolint:wrapcheck // the tests read the sentinel.
}

func (s *testSuite) TestAnOperatorsAdminSessionIsAnsweredAsItsAccount() {
	s.player(ada, "game-token")
	s.operates(ada, true)
	s.adminSession(ada, "admin-token")

	for _, header := range []string{"cp_admin=admin-token", "cp_sid=game-token; cp_admin=admin-token"} {
		account, err := s.operator(header, start)

		s.Require().NoError(err, header)
		s.Equal(ada.String(), account, header)
	}
}

func (s *testSuite) TestTheGameSessionNeverAnswersForAnOperator() {
	s.player(ada, "game-token")
	s.operates(ada, true)

	for _, header := range []string{"cp_sid=game-token", "cp_admin=game-token"} {
		_, err := s.operator(header, start)
		s.Require().ErrorIs(err, operator_query.ErrNoSession, header)
	}
}

func (s *testSuite) TestUnflaggingAnOperatorEndsTheSessionItHolds() {
	s.player(ada, "game-token")
	s.operates(ada, true)
	s.adminSession(ada, "admin-token")

	s.operates(ada, false)

	_, err := s.operator("cp_admin=admin-token", start)
	s.Require().ErrorIs(err, operator_query.ErrNotAnOperator)
}

func (s *testSuite) TestAnAdminSessionEndsWithItsLifetimeOrItsSignOut() {
	s.player(ada, "game-token")
	s.operates(ada, true)
	expiresAt := s.adminSession(ada, "admin-token")

	_, err := s.operator("cp_admin=admin-token", expiresAt.Add(-time.Microsecond))
	s.Require().NoError(err, "just before it ends")
	_, err = s.operator("cp_admin=admin-token", expiresAt)
	s.Require().ErrorIs(err, operator_query.ErrNoSession, "never extended")

	s.Require().NoError(s.sessions.DeleteSession(s.T().Context(), accounts.TokenOf("admin-token").Hash()))
	_, err = s.operator("cp_admin=admin-token", start)
	s.Require().ErrorIs(err, operator_query.ErrNoSession, "signed out")
}

func (s *testSuite) TestAGameSignOutKeepsTheAdminSession() {
	s.player(ada, "game-token")
	s.operates(ada, true)
	s.adminSession(ada, "admin-token")

	s.Require().NoError(s.accounts.DeleteSession(s.T().Context(), accounts.TokenOf("game-token").Hash()))

	_, err := s.accounts.Session(s.T().Context(), accounts.TokenOf("game-token").Hash())
	s.Require().ErrorIs(err, accounts.ErrSessionNotFound)
	account, err := s.operator("cp_sid=game-token; cp_admin=admin-token", start)
	s.Require().NoError(err)
	s.Equal(ada.String(), account)
}

func (s *testSuite) TestNoCookieIsNobody() {
	for _, header := range []string{"", "cp_admin=", "cp_admin=made-up", ";;;=", "CP_ADMIN=admin-token"} {
		_, err := s.operator(header, start)
		s.Require().ErrorIs(err, operator_query.ErrNoSession, header)
	}
}
