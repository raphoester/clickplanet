package accounts_query_test

import (
	"time"

	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_account_handler/account_query"
)

func (s *testSuite) TestGetAccountsAnswersWhatGetAccountAndTheClickTokenSay() {
	cara, dan := accounts.AccountID{15: 3}, accounts.AccountID{15: 4}
	s.guest(ada, "token-1", start)
	s.signUp(bob, "google", "token-2", start.Add(time.Minute))
	s.guest(cara, "token-3", start.Add(time.Hour))
	identity := accounts.NewIdentity("email", accounts.ClaimOf("cara@example.com", "cara@example.com", true), cara, start.Add(2*time.Hour))
	session := accounts.LinkedSession(cara, accounts.TokenOf("token-4"), lifetime, start.Add(2*time.Hour))
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), accounts.NewSignIn(session).WithIdentity(identity)))
	s.guest(dan, "token-5", start)
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), dan))

	tokens := map[accounts.AccountID]string{ada: "token-1", bob: "token-2", cara: "token-3"}
	page := s.accounts(ada, bob, cara, dan)

	s.Len(page.GetAccounts(), len(tokens))
	for _, found := range page.GetAccounts() {
		id, err := accounts.AccountIDOf(found.GetAccountId())
		s.Require().NoError(err)

		one, err := account_query.NewPostgresQuery(s.db).Account(s.T().Context(), id)
		s.Require().NoError(err)
		s.True(proto.Equal(&authv1.GetAccountResponse{Linked: found.GetLinked(), CreatedAtUnixMs: found.GetCreatedAtUnixMs()}, one),
			"%s: %v against %v", id, found, one)

		minted, err := s.store.Session(s.T().Context(), accounts.TokenOf(tokens[id]).Hash())
		s.Require().NoError(err)
		s.Equal(minted.Linked(), found.GetLinked(), "%s: the click token says linked exactly when GetAccounts does", id)
	}

	gone, err := account_query.NewPostgresQuery(s.db).Account(s.T().Context(), dan)
	s.Require().NoError(err)
	s.Empty(gone.String(), "a deleted account is left out of both")
}
