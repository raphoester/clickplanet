package caller_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_caller_handler/caller_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
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
	start    = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{GuestTTL: 90 * 24 * time.Hour, ExtendEvery: 24 * time.Hour}
	ada      = accounts.AccountID{15: 1}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_account_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(s.store.CreateGuest(s.T().Context(),
		accounts.GuestSession(ada, accounts.TokenOf("token-1"), lifetime, start)))
}

func (s *testSuite) callerAt(now time.Time, cookieHeader string) string {
	answer, err := caller_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now)).Caller(s.T().Context(), cookieHeader)
	s.Require().NoError(err)
	return answer.GetAccountId()
}

func (s *testSuite) TestALiveSessionNamesItsAccount() {
	s.Equal(ada.String(), s.callerAt(start.Add(89*24*time.Hour), "theme=dark; cp_sid=token-1"))
}

func (s *testSuite) TestASessionPastItsExpiryNamesNobody() {
	s.Empty(s.callerAt(start.Add(90*24*time.Hour), "cp_sid=token-1"), "expired at its expiry")
	s.Empty(s.callerAt(start.Add(91*24*time.Hour), "cp_sid=token-1"))
}

func (s *testSuite) TestACookieWithNoSessionNamesNobody() {
	for _, header := range []string{"cp_sid=made-up", "", "theme=dark", "cp_sid=", "cp_sid"} {
		s.Empty(s.callerAt(start, header), "%q", header)
	}
}
