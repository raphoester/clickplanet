package email_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_email_handler/email_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
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
	start    = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	ada      = accounts.AccountID{15: 1}
)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_account_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(s.store.CreateGuest(s.T().Context(), accounts.GuestSession(ada, accounts.TokenOf("token-1"), lifetime, start)))
}

func (s *testSuite) link(provider string, claim accounts.Claim, at time.Time) {
	identity := accounts.NewIdentity(provider, claim, ada, at)
	session := accounts.LinkedSession(ada, accounts.TokenOf("token-"+provider), lifetime, at)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), accounts.NewSignIn(session).WithIdentity(identity)))
}

func (s *testSuite) email(account accounts.AccountID) *authv1.GetEmailResponse {
	answer, err := email_query.NewPostgresQuery(s.db).Email(s.T().Context(), account)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestAGuestAndAnUnknownAccountHoldNoAddress() {
	s.Nil(s.email(ada).Email)
	s.Nil(s.email(accounts.AccountID{15: 9}).Email)
}

func (s *testSuite) TestAnUnverifiedAddressIsNoAddress() {
	s.link("discord", accounts.ClaimOf("discord-user", "ada@example.com", false), start)

	s.Nil(s.email(ada).Email)
}

func (s *testSuite) TestTheOldestVerifiedAddressIsTheOneToWriteTo() {
	s.link("discord", accounts.ClaimOf("discord-user", "", false), start)
	s.link("google", accounts.ClaimOf("google-user", "Ada@Example.com", true), start.Add(time.Hour))
	s.link("email", accounts.ClaimOf("ada+game@example.org", "ada+game@example.org", true), start.Add(2*time.Hour))

	want := "Ada@Example.com"
	s.True(proto.Equal(&authv1.GetEmailResponse{Email: &want}, s.email(ada)), "got %v", s.email(ada))
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := email_query.NewPostgresQuery(s.db).Email(ctx, ada)

	s.Error(err, "not an answer with no address")
}
