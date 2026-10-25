package accounts_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_accounts_handler/accounts_query"
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
	bob      = accounts.AccountID{15: 2}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_account_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) guest(account accounts.AccountID, token string, at time.Time) {
	s.Require().NoError(s.store.CreateGuest(s.T().Context(), accounts.GuestSession(account, accounts.TokenOf(token), lifetime, at)))
}

func (s *testSuite) signUp(account accounts.AccountID, provider string, token string, at time.Time) {
	identity := accounts.NewIdentity(provider, accounts.ClaimOf(provider+"-user", "", false), account, at)
	session := accounts.LinkedSession(account, accounts.TokenOf(token), lifetime, at)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), accounts.NewSignIn(session).WithNewAccount().WithIdentity(identity)))
}

func (s *testSuite) accounts(asked ...accounts.AccountID) *authv1.GetAccountsResponse {
	answer, err := accounts_query.NewPostgresQuery(s.db).Accounts(s.T().Context(), asked)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestTheKnownAccountsAreAnsweredInIDOrderAndTheOthersLeftOut() {
	s.signUp(bob, "google", "token-2", start)
	s.guest(ada, "token-1", start.Add(time.Hour))

	answer := s.accounts(bob, accounts.AccountID{15: 9}, ada)

	s.True(proto.Equal(&authv1.GetAccountsResponse{Accounts: []*authv1.Account{
		{AccountId: ada.String(), CreatedAtUnixMs: start.Add(time.Hour).UnixMilli()},
		{AccountId: bob.String(), Linked: true, CreatedAtUnixMs: start.UnixMilli()},
	}}, answer), "got %v", answer)
}

func (s *testSuite) TestAnAccountAskedTwiceIsAnsweredOnce() {
	s.guest(ada, "token-1", start)

	s.Len(s.accounts(ada, ada).GetAccounts(), 1)
}

func (s *testSuite) TestNoAccountAskedIsNoAccount() {
	s.guest(ada, "token-1", start)

	s.Empty(s.accounts().GetAccounts())
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := accounts_query.NewPostgresQuery(s.db).Accounts(ctx, []accounts.AccountID{ada})

	s.Error(err)
}
