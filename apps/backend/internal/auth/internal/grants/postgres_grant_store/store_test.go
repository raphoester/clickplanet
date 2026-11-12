package postgres_grant_store_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants/aes_grant_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants/postgres_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	grants.StoreContractSuite

	db     *cppg.Postgres
	sealer *aes_grant_sealer.Sealer
	store  *postgres_grant_store.Store
}

var at = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	sealer, err := aes_grant_sealer.New(bytes.Repeat([]byte{1}, 32))
	s.Require().NoError(err)
	s.sealer = sealer
	s.store = postgres_grant_store.New(s.db, sealer)
	s.NewStore = func() grants.Store { return s.store }
	s.Kept = func(grants.Store) []grants.Snapshot { return s.kept() }
	s.NewAccount = s.account
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

func (s *testSuite) account(account grants.AccountID) {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(),
		`INSERT INTO accounts (id, created_at, last_seen_at) VALUES ($1, $2, $2) ON CONFLICT DO NOTHING`, uuid.UUID(account), at)
	s.Require().NoError(err)
}

func (s *testSuite) kept() []grants.Snapshot {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(),
		`SELECT grant_id, account_id, discord_user_id, sealed_token, expires_at FROM discord_grants ORDER BY grant_id`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var kept []grants.Snapshot
	for rows.Next() {
		var (
			id, account uuid.UUID
			user        string
			sealed      []byte
			expiresAt   time.Time
		)
		s.Require().NoError(rows.Scan(&id, &account, &user, &sealed, &expiresAt))
		token, err := s.sealer.Opened(grants.GrantID(id), sealed)
		s.Require().NoError(err)
		kept = append(kept, grants.Snapshot{
			ID: grants.GrantID(id), Account: grants.AccountID(account), User: user, Token: token, ExpiresAt: expiresAt,
		})
	}
	s.Require().NoError(rows.Err())
	return kept
}

func (s *testSuite) save(id byte, account byte) grants.Grant {
	s.account(grants.AccountID{15: account})
	grant, err := grants.NewGrant(grants.GrantID{15: id}, grants.AccountID{15: account}, "4242", "the-access-token", at)
	s.Require().NoError(err)
	s.Require().NoError(s.store.SaveGrant(s.T().Context(), grant))
	return grant
}

func (s *testSuite) TestTheTokenIsSealedAtRest() {
	s.save(1, 1)

	var sealed []byte
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(),
		`SELECT sealed_token FROM discord_grants`).Scan(&sealed))
	s.NotEmpty(sealed)
	s.NotContains(string(sealed), "the-access-token")
}

func (s *testSuite) TestATokenMovedToAnotherGrantDoesNotOpen() {
	s.save(1, 1)
	s.save(2, 1)
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `
		UPDATE discord_grants SET sealed_token = (SELECT sealed_token FROM discord_grants WHERE grant_id = $1)
		WHERE grant_id = $2
	`, uuid.UUID(grants.GrantID{15: 1}), uuid.UUID(grants.GrantID{15: 2}))
	s.Require().NoError(err)

	_, err = s.store.Grant(s.T().Context(), grants.GrantID{15: 2})

	s.Require().ErrorIs(err, aes_grant_sealer.ErrNotSealedHere)
	s.NotErrorIs(err, grants.ErrGrantNotFound, "a broken seal is a failure, not a grant nobody kept")
}

func (s *testSuite) TestADeletedAccountTakesItsGrants() {
	s.save(1, 1)
	kept := s.save(2, 2)

	s.Require().NoError(postgres_account_store.New(s.db).DeleteAccount(s.T().Context(), grants.AccountID{15: 1}))

	s.Require().Len(s.kept(), 1)
	s.Equal(kept.ID(), s.kept()[0].ID)
}

func (s *testSuite) TestAGrantNeedsAnAccountTheSchemaKeeps() {
	grant, err := grants.NewGrant(grants.GrantID{15: 1}, grants.AccountID{15: 9}, "4242", "the-access-token", at)
	s.Require().NoError(err)

	s.Error(s.store.SaveGrant(s.T().Context(), grant))
}
