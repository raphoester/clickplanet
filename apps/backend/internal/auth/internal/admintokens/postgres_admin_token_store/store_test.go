package postgres_admin_token_store_test

import (
	"encoding/hex"
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

func (s *testSuite) issued(at time.Time) *admintokens.Pair {
	pair, err := s.issuer.Pair(claude, at)
	s.Require().NoError(err)
	s.Require().NoError(s.store.SaveTokens(s.T().Context(), pair.TakeSnapshots()))
	return pair
}

func (s *testSuite) kinds() map[string]string {
	rows, err := s.db.QueryContext(s.T().Context(), `SELECT encode(token_hash, 'hex'), kind::text FROM admin_tokens`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	kept := map[string]string{}
	for rows.Next() {
		var hash, kind string
		s.Require().NoError(rows.Scan(&hash, &kind))
		kept[hash] = kind
	}
	s.Require().NoError(rows.Err())
	return kept
}

func hexOf(secret admintokens.Secret) string {
	return hex.EncodeToString(secret.Hash())
}

func (s *testSuite) TestAPairIsKeptAsTheHashesOfItsTwoTokens() {
	pair := s.issued(start)

	s.Equal(map[string]string{hexOf(pair.Access()): "access", hexOf(pair.Refresh()): "refresh"}, s.kinds())
}

func (s *testSuite) TestARefreshTokenIsSpentOnceAndNamesItsHolder() {
	pair := s.issued(start)

	holder, err := s.store.SpendRefresh(s.T().Context(), pair.Refresh().Hash(), start.Add(time.Hour))
	s.Require().NoError(err)
	s.Equal(claude, holder)

	_, err = s.store.SpendRefresh(s.T().Context(), pair.Refresh().Hash(), start.Add(time.Hour))
	s.Require().ErrorIs(err, admintokens.ErrUnknownToken, "spent")
	s.Equal(map[string]string{hexOf(pair.Access()): "access"}, s.kinds())
}

func (s *testSuite) TestAnAccessTokenOrALapsedOneBuysNothing() {
	pair := s.issued(start)

	_, err := s.store.SpendRefresh(s.T().Context(), pair.Access().Hash(), start)
	s.Require().ErrorIs(err, admintokens.ErrUnknownToken, "an access token is not a refresh token")

	_, err = s.store.SpendRefresh(s.T().Context(), pair.Refresh().Hash(), start.Add(admintokens.RefreshLifetime))
	s.Require().ErrorIs(err, admintokens.ErrUnknownToken, "lapsed")
}

func (s *testSuite) TestTheLapsedTokensGoAndTheLiveOnesStay() {
	old := s.issued(start)
	fresh := s.issued(start.Add(2 * time.Hour))

	s.Require().NoError(s.store.DeleteLapsed(s.T().Context(), start.Add(2*time.Hour)))

	s.Equal(map[string]string{
		hexOf(old.Refresh()): "refresh", hexOf(fresh.Access()): "access", hexOf(fresh.Refresh()): "refresh",
	}, s.kinds())
}

func (s *testSuite) TestATokenGoesWithItsAccount() {
	s.issued(start)

	_, err := s.db.ExecContext(s.T().Context(), `DELETE FROM accounts WHERE id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)

	s.Empty(s.kinds())
}
