package token_operator_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens/postgres_admin_token_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_token_operator_handler/token_operator_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db     *cppg.Postgres
	tokens *postgres_admin_token_store.Store
}

var (
	start  = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	ada    = accounts.AccountID{15: 1}
	claude = admintokens.Holder{Account: ada, ClientID: "the client", Client: "Claude"}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.tokens = postgres_admin_token_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(postgres_account_store.New(s.db).CreateGuest(s.T().Context(),
		accounts.GuestSession(ada, accounts.TokenOf("game-token"), accounts.Lifetime{}.WithDefaults(), start)))
}

func (s *testSuite) operates(operates bool) {
	_, err := s.db.ExecContext(s.T().Context(), `UPDATE accounts SET operator = $2 WHERE id = $1`, ada.String(), operates)
	s.Require().NoError(err)
}

// Issued as the domain issues one, so the hash the store keeps is the one this query computes.
func (s *testSuite) issued() *admintokens.Pair {
	pair, err := admintokens.NewIssuer(&admintokens.SequentialSecrets{}).Pair(claude, start)
	s.Require().NoError(err)
	s.Require().NoError(s.tokens.SaveTokens(s.T().Context(), pair.TakeSnapshots()))
	return pair
}

func (s *testSuite) operator(token admintokens.Secret, at time.Time) (*authv1.GetTokenOperatorResponse, error) {
	return token_operator_query.NewPostgresQuery(s.db, cptime.NewFixedClock(at)).Operator(s.T().Context(), string(token)) //nolint:wrapcheck // the tests read the sentinel.
}

func (s *testSuite) TestAnOperatorsAccessTokenIsAnsweredWithItsAccountAndItsApp() {
	s.operates(true)
	pair := s.issued()

	answer, err := s.operator(pair.Access(), start)

	s.Require().NoError(err)
	s.Equal(&authv1.GetTokenOperatorResponse{AccountId: ada.String(), Client: "Claude"}, answer)
}

func (s *testSuite) TestAnAccessTokenEndsWithItsHour() {
	s.operates(true)
	pair := s.issued()

	_, err := s.operator(pair.Access(), start.Add(admintokens.AccessLifetime-time.Microsecond))
	s.Require().NoError(err, "just before it ends")
	_, err = s.operator(pair.Access(), start.Add(admintokens.AccessLifetime))
	s.Require().ErrorIs(err, token_operator_query.ErrNoToken, "never extended")
}

func (s *testSuite) TestARefreshTokenOpensNothing() {
	s.operates(true)
	pair := s.issued()

	_, err := s.operator(pair.Refresh(), start)

	s.Require().ErrorIs(err, token_operator_query.ErrNoToken)
}

func (s *testSuite) TestUnflaggingAnOperatorEndsEveryConnectionItHolds() {
	s.operates(true)
	pair := s.issued()

	s.operates(false)

	_, err := s.operator(pair.Access(), start)
	s.Require().ErrorIs(err, token_operator_query.ErrNotAnOperator)
}

func (s *testSuite) TestATokenNobodyHoldsIsNobody() {
	s.operates(true)
	pair := s.issued()

	for _, token := range []admintokens.Secret{"", "cpat_made-up", "game-token", pair.Access() + " ", "Bearer " + pair.Access()} {
		_, err := s.operator(token, start)
		s.Require().ErrorIs(err, token_operator_query.ErrNoToken, token)
	}
}
