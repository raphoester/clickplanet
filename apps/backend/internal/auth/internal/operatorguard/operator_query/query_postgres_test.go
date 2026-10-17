package operator_query_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/operatorguard/operator_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_account_store.Store
}

var (
	start    = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	ada      = accounts.AccountID{15: 1}
	bob      = accounts.AccountID{15: 2}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_account_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) guest(account accounts.AccountID, token string) {
	session := accounts.GuestSession(account, accounts.TokenOf(token), lifetime, start)
	s.Require().NoError(s.store.CreateGuest(s.T().Context(), session))
}

func (s *testSuite) link(account accounts.AccountID, subject string, token string) {
	identity := accounts.NewIdentity("google", accounts.ClaimOf(subject, "", false), account, start)
	session := accounts.LinkedSession(account, accounts.TokenOf(token), lifetime, start)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), accounts.NewSignIn(session).WithIdentity(identity)))
}

func (s *testSuite) operates(account accounts.AccountID) {
	_, err := s.db.ExecContext(s.T().Context(), `UPDATE accounts SET role = 'operator' WHERE id = $1`, account.String())
	s.Require().NoError(err)
}

func (s *testSuite) operator(cookieHeader string, at time.Time) (string, error) {
	return operator_query.NewPostgresQuery(s.db, cptime.NewFixedClock(at)).Operator(s.T().Context(), cookieHeader) //nolint:wrapcheck // the tests read the sentinel.
}

func (s *testSuite) TestASignedInOperatorIsAnsweredAsItsAccount() {
	s.guest(ada, "token-1")
	s.link(ada, "ada", "token-2")
	s.operates(ada)

	for _, header := range []string{"cp_sid=token-1", "theme=dark; cp_sid=token-2"} {
		account, err := s.operator(header, start)

		s.Require().NoError(err, header)
		s.Equal(ada.String(), account, header)
	}
}

func (s *testSuite) TestASignedInPlayerIsNotAnOperator() {
	s.guest(ada, "token-1")
	s.link(ada, "ada", "token-2")

	_, err := s.operator("cp_sid=token-2", start)

	s.Require().ErrorIs(err, operator_query.ErrNotAnOperator)
}

func (s *testSuite) TestAGuestIsNeverAnOperator() {
	s.guest(ada, "token-1")
	s.operates(ada)

	_, err := s.operator("cp_sid=token-1", start)

	s.Require().ErrorIs(err, operator_query.ErrNotAnOperator, "a guest's cookie is the whole of what proves it")
}

func (s *testSuite) TestNoLiveSessionIsNobody() {
	s.guest(ada, "token-1")
	s.link(ada, "ada", "token-2")
	s.operates(ada)

	for _, header := range []string{"", "cp_sid=", "cp_sid=made-up", ";;;=", "CP_SID=token-2"} {
		_, err := s.operator(header, start)
		s.Require().ErrorIs(err, operator_query.ErrNoSession, header)
	}

	_, err := s.operator("cp_sid=token-2", start.Add(lifetime.LinkedTTL))
	s.Require().ErrorIs(err, operator_query.ErrNoSession, "an expired session")
}

func (s *testSuite) TestTheOperatorQueryFindsTheCallerTheCommandsFind() {
	s.guest(ada, "token-1")
	s.link(ada, "ada", "token-2")
	s.operates(ada)
	s.guest(bob, "token-3")
	s.link(bob, "bob", "token-4")
	cara := accounts.AccountID{15: 3}
	s.guest(cara, "token-5")
	s.operates(cara)
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), cara))

	headers := []string{
		"cp_sid=token-1", "cp_sid=token-2", "cp_sid=token-3", "cp_sid=token-4", "cp_sid=token-5",
		"", "cp_sid=made-up", "cp_sid=token-3; cp_sid=token-2", "cp_sid=; cp_sid=token-2",
	}
	expiry := s.kept("token-2").ExpiresAt()
	times := []time.Time{start, expiry.Add(-time.Nanosecond), expiry, expiry.Add(time.Nanosecond)}

	admitted := 0
	for _, header := range headers {
		for _, at := range times {
			name := fmt.Sprintf("%q at %s", header, at.Format(time.RFC3339Nano))
			session, err := accounts.Caller(s.T().Context(), s.store, header, at)
			account, queryErr := s.operator(header, at)

			if errors.Is(err, accounts.ErrNoAccount) {
				s.Require().ErrorIs(queryErr, operator_query.ErrNoSession, name)
				continue
			}
			s.Require().NoError(err, name)

			if session.Account() != ada {
				s.Require().ErrorIs(queryErr, operator_query.ErrNotAnOperator, name)
				continue
			}
			s.Require().NoError(queryErr, name)
			s.Equal(ada.String(), account, name)
			admitted++
		}
	}
	s.Positive(admitted, "the operator is admitted while its session lives")
}

func (s *testSuite) kept(token string) *accounts.Session {
	session, err := s.store.Session(s.T().Context(), accounts.TokenOf(token).Hash())
	s.Require().NoError(err)
	return session
}
