package account_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_account_handler/account_query"
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

var (
	start    = time.Date(2026, 9, 17, 12, 0, 0, 123_456_000, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	ada      = accounts.AccountID{15: 1}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_account_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(s.store.CreateGuest(s.T().Context(), accounts.GuestSession(ada, accounts.TokenOf("token-1"), lifetime, start)))
}

func (s *testSuite) account(account accounts.AccountID) *authv1.GetAccountResponse {
	answer, err := account_query.NewPostgresQuery(s.db).Account(s.T().Context(), account)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestAGuestIsNotLinkedAndSaysWhenItWasMadeToTheMillisecond() {
	s.True(proto.Equal(&authv1.GetAccountResponse{CreatedAtUnixMs: start.UnixMilli()}, s.account(ada)), "got %v", s.account(ada))
}

func (s *testSuite) TestAnAccountWithAnIdentityIsLinkedAndKeepsTheDayItWasMade() {
	identity := accounts.NewIdentity("discord", accounts.ClaimOf("discord-user", "", false), ada, start.Add(time.Hour))
	session := accounts.LinkedSession(ada, accounts.TokenOf("token-2"), lifetime, start.Add(time.Hour))
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), accounts.NewSignIn(session).WithIdentity(identity)))

	s.True(proto.Equal(&authv1.GetAccountResponse{Linked: true, CreatedAtUnixMs: start.UnixMilli()}, s.account(ada)),
		"got %v", s.account(ada))
}

func (s *testSuite) TestAnUnknownOrDeletedAccountIsAnEmptyAnswer() {
	s.Empty(s.account(accounts.AccountID{15: 9}).String())

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), ada))
	s.Empty(s.account(ada).String())
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := account_query.NewPostgresQuery(s.db).Account(ctx, ada)

	s.Error(err, "not an empty answer")
}
