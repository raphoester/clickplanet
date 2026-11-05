package postgres_admin_token_store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens/postgres_admin_token_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db     *cppg.Postgres
	store  *postgres_admin_token_store.Store
	issuer *admintokens.Issuer
}

var (
	start  = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	ada    = accounts.AccountID{15: 1}
	claude = admintokens.Holder{Account: ada, ClientID: "the client", Client: "Claude"}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_admin_token_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(postgres_account_store.New(s.db).CreateGuest(s.T().Context(),
		accounts.GuestSession(ada, accounts.TokenOf("guest"), accounts.Lifetime{}.WithDefaults(), start)))
	s.issuer = admintokens.NewIssuer(&admintokens.SequentialSecrets{})
}

func (s *testSuite) connected(at time.Time) *admintokens.Pair {
	pair, err := s.issuer.Connected(claude, at)
	s.Require().NoError(err)
	s.Require().NoError(s.store.SaveTokens(s.T().Context(), pair.Tokens()...))
	return pair
}

func (s *testSuite) connectionOf(secret admintokens.Secret) []admintokens.Snapshot {
	connection, err := s.store.Connection(s.T().Context(), secret.Hash())
	s.Require().NoError(err)
	kept := make([]admintokens.Snapshot, 0, len(connection))
	for _, token := range connection {
		kept = append(kept, token.TakeSnapshot())
	}
	return kept
}

func (s *testSuite) TestAConnectionIsReadBackAsItWasSaved() {
	pair := s.connected(start)
	s.connected(start)

	s.ElementsMatch(pair.Tokens(), s.connectionOf(pair.TakeSnapshot().Access))
	s.ElementsMatch(pair.Tokens(), s.connectionOf(pair.TakeSnapshot().Refresh), "from either of its tokens, and none of another")
}

func (s *testSuite) TestASavedTokenReplacesTheOneKeptUnderItsHash() {
	pair := s.connected(start)
	spent := pair.Tokens()[1]
	spent.Kind = admintokens.KindSpent
	spent.ExpiresAt = start.Add(time.Minute)

	s.Require().NoError(s.store.SaveTokens(s.T().Context(), spent))

	s.ElementsMatch([]admintokens.Snapshot{pair.Tokens()[0], spent}, s.connectionOf(pair.TakeSnapshot().Access))
}

func (s *testSuite) TestATokenNobodyHoldsIsNoConnection() {
	s.connected(start)

	s.Empty(s.connectionOf("cprt_made-up"))
}

func (s *testSuite) TestATokenGoesWithItsAccount() {
	pair := s.connected(start)

	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `DELETE FROM accounts WHERE id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)

	s.Empty(s.connectionOf(pair.TakeSnapshot().Access))
}
