package emails_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/search_emails_handler/emails_query"
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
	start    = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	ada      = accounts.AccountID{15: 1}
	bob      = accounts.AccountID{15: 2}
	cara     = accounts.AccountID{15: 3}
	dan      = accounts.AccountID{15: 4}
)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_account_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) signUp(account accounts.AccountID, provider string, claim accounts.Claim, at time.Time) {
	identity := accounts.NewIdentity(provider, claim, account, at)
	session := accounts.LinkedSession(account, accounts.TokenOf("token-"+account.String()), lifetime, at)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), accounts.NewSignIn(session).WithNewAccount().WithIdentity(identity)))
}

func (s *testSuite) code(account accounts.AccountID, address string, at time.Time) {
	s.signUp(account, "email", accounts.ClaimOf(address, address, true), at)
}

func (s *testSuite) holders(text string, limit int, offset int) *authv1.SearchEmailsResponse {
	answer, err := emails_query.NewPostgresQuery(s.db).Holders(s.T().Context(), text, limit, offset)
	s.Require().NoError(err)
	return answer
}

func holder(account accounts.AccountID, email string, provider string, at time.Time) *authv1.EmailHolder {
	return &authv1.EmailHolder{AccountId: account.String(), Email: email, Provider: provider, LinkedAt: timestamppb.New(at)}
}

func (s *testSuite) TestEveryVerifiedAddressHoldingTheTextIsAnsweredInAnyCaseOrderedByAddress() {
	s.code(cara, "john+2@gmail.com", start.Add(2*time.Hour))
	s.code(bob, "john+1@gmail.com", start.Add(time.Hour))
	s.signUp(ada, "google", accounts.ClaimOf("google-ada", "John+0@Gmail.com", true), start)
	s.signUp(dan, "discord", accounts.ClaimOf("4444", "john+3@gmail.com", false), start)

	answer := s.holders("JOHN+", 10, 0)

	s.True(proto.Equal(&authv1.SearchEmailsResponse{Holders: []*authv1.EmailHolder{
		holder(ada, "John+0@Gmail.com", "google", start),
		holder(bob, "john+1@gmail.com", "email", start.Add(time.Hour)),
		holder(cara, "john+2@gmail.com", "email", start.Add(2*time.Hour)),
	}}, answer), "an address Discord did not verify is not kept, so not found: got %v", answer)
}

func (s *testSuite) TestOneAddressHeldTwiceIsAnsweredOldestLinkFirst() {
	s.code(bob, "ada@proton.me", start.Add(time.Hour))
	s.signUp(ada, "discord", accounts.ClaimOf("1111", "ada@proton.me", true), start)

	answer := s.holders("ada@", 10, 0)

	s.True(proto.Equal(&authv1.SearchEmailsResponse{Holders: []*authv1.EmailHolder{
		holder(ada, "ada@proton.me", "discord", start),
		holder(bob, "ada@proton.me", "email", start.Add(time.Hour)),
	}}, answer), "got %v", answer)
}

func (s *testSuite) TestAnUnderscoreOrAPercentIsMatchedAsItself() {
	s.code(ada, "a_b@gmail.com", start)
	s.code(bob, "axb@gmail.com", start)
	s.code(cara, "100%@gmail.com", start)

	s.True(proto.Equal(&authv1.SearchEmailsResponse{Holders: []*authv1.EmailHolder{
		holder(ada, "a_b@gmail.com", "email", start),
	}}, s.holders("a_b", 10, 0)))
	s.True(proto.Equal(&authv1.SearchEmailsResponse{Holders: []*authv1.EmailHolder{
		holder(cara, "100%@gmail.com", "email", start),
	}}, s.holders("%", 10, 0)))
}

func (s *testSuite) TestAPageSkipsTheOffsetAndStopsAtTheLimit() {
	s.code(ada, "a@gmail.com", start)
	s.code(bob, "b@gmail.com", start)
	s.code(cara, "c@gmail.com", start)

	s.True(proto.Equal(&authv1.SearchEmailsResponse{Holders: []*authv1.EmailHolder{
		holder(bob, "b@gmail.com", "email", start),
	}}, s.holders("@gmail", 1, 1)))
	s.Empty(s.holders("@gmail", 10, 3).GetHolders(), "a page past the last match is empty")
}
