package discord_users_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_discord_users_handler/discord_users_query"
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
	start    = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	ada      = accounts.AccountID{15: 1}
	bob      = accounts.AccountID{15: 2}
	cara     = accounts.AccountID{15: 3}
	dan      = accounts.AccountID{15: 4}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_account_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) signUp(account accounts.AccountID, provider string, subject string) {
	identity := accounts.NewIdentity(provider, accounts.ClaimOf(subject, "", false), account, start)
	session := accounts.LinkedSession(account, accounts.TokenOf("token-"+account.String()), lifetime, start)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), accounts.NewSignIn(session).WithNewAccount().WithIdentity(identity)))
}

func (s *testSuite) guest(account accounts.AccountID) {
	s.Require().NoError(s.store.CreateGuest(s.T().Context(),
		accounts.GuestSession(account, accounts.TokenOf("token-"+account.String()), lifetime, start)))
}

func (s *testSuite) users(after accounts.AccountID, limit int) *authv1.GetDiscordUsersResponse {
	answer, err := discord_users_query.NewPostgresQuery(s.db).DiscordUsers(s.T().Context(), after, limit)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestTheDiscordAccountsAreAnsweredInIDOrderWithTheirUser() {
	s.signUp(cara, "discord", "3333")
	s.signUp(ada, "discord", "1111")
	s.signUp(bob, "google", "google-bob")
	s.guest(dan)

	answer := s.users(accounts.AccountID{}, 10)

	s.True(proto.Equal(&authv1.GetDiscordUsersResponse{Users: []*authv1.DiscordUser{
		{AccountId: ada.String(), DiscordUserId: "1111"},
		{AccountId: cara.String(), DiscordUserId: "3333"},
	}}, answer), "got %v", answer)
}

func (s *testSuite) TestAPageStartsAfterTheAccountAndStopsAtTheLimit() {
	s.signUp(ada, "discord", "1111")
	s.signUp(bob, "discord", "2222")
	s.signUp(cara, "discord", "3333")

	answer := s.users(ada, 1)

	s.True(proto.Equal(&authv1.GetDiscordUsersResponse{Users: []*authv1.DiscordUser{
		{AccountId: bob.String(), DiscordUserId: "2222"},
	}}, answer), "got %v", answer)
}

func (s *testSuite) TestAnAccountAskedDeletedIsLeftOut() {
	s.signUp(ada, "discord", "1111")
	s.signUp(bob, "discord", "2222")
	s.Require().NoError(s.store.RequestDeletion(s.T().Context(), ada, start.Add(time.Hour)))

	answer := s.users(accounts.AccountID{}, 10)

	s.True(proto.Equal(&authv1.GetDiscordUsersResponse{Users: []*authv1.DiscordUser{
		{AccountId: bob.String(), DiscordUserId: "2222"},
	}}, answer), "got %v", answer)
}

func (s *testSuite) TestNoDiscordAccountIsAnEmptyPage() {
	s.signUp(ada, "google", "google-ada")

	s.Empty(s.users(accounts.AccountID{}, 10).GetUsers())
}
