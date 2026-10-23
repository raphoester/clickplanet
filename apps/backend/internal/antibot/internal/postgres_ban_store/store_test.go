package postgres_ban_store_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_ban_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "antibot", migrations.FS)
}

func (s *testSuite) purge() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) TestScopesKeepTheContract() {
	suite.Run(s.T(), &bans.StoreContractSuite{
		NewStore: func() bans.Store { s.purge(); return postgres_ban_store.NewScopes(s.db) },
		Key:      func(n int) string { return fmt.Sprintf("2001:db8:%x::/64", n) },
	})
}

func (s *testSuite) TestAccountsKeepTheContract() {
	suite.Run(s.T(), &bans.StoreContractSuite{
		NewStore: func() bans.Store { s.purge(); return postgres_ban_store.NewAccounts(s.db) },
		Key:      func(n int) string { return fmt.Sprintf("0b7e5b6c-8f3a-4d2e-9c1a-%012x", n) },
	})
}

var until = time.Date(2026, 10, 8, 12, 30, 0, 0, time.UTC)

func (s *testSuite) TestAccountsAreKeptApartFromScopes() {
	s.purge()
	ctx := s.T().Context()
	scopes, accounts := postgres_ban_store.NewScopes(s.db), postgres_ban_store.NewAccounts(s.db)
	const account = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

	s.Require().NoError(accounts.Change(ctx, account, func(record bans.Record, _ bool) (bans.Record, bool) {
		record.Offences, record.ExpiresAt = 1, until
		return record, true
	}))

	_, found, err := scopes.Record(ctx, account)
	s.Require().NoError(err)
	s.False(found)

	running, err := scopes.Running(ctx, until.Add(-time.Hour))
	s.Require().NoError(err)
	s.Zero(running)
}

func (s *testSuite) TestAKeyThatIsNotAnAccountIsAnError() {
	s.purge()
	accounts := postgres_ban_store.NewAccounts(s.db)

	s.Require().Error(accounts.Change(s.T().Context(), "1.2.3.4", func(record bans.Record, _ bool) (bans.Record, bool) {
		return record, true
	}))

	running, err := accounts.Running(s.T().Context(), until)
	s.Require().NoError(err)
	s.Zero(running)
}

func (s *testSuite) TestEveryPardonIsKept() {
	s.purge()
	ctx := s.T().Context()
	const (
		scope   = "2001:db8:1::/64"
		account = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	)
	keys := map[string]*postgres_ban_store.Store{
		scope:   postgres_ban_store.NewScopes(s.db),
		account: postgres_ban_store.NewAccounts(s.db),
	}

	for key, store := range keys {
		for i := range 2 {
			at := until.Add(time.Duration(i) * time.Hour)
			s.Require().NoError(store.Change(ctx, key, func(record bans.Record, _ bool) (bans.Record, bool) {
				record.ExpiresAt = at
				record.Pardon = bans.Pardon{At: at, Lifted: bans.Sentence{Offence: 1, Until: at}, Until: at.Add(time.Hour)}
				return record, true
			}))
		}
	}

	var pardons, accountPardons int
	s.Require().NoError(s.db.QueryRowContext(ctx, `SELECT count(*) FROM pardons WHERE scope = $1`, scope).Scan(&pardons))
	s.Require().NoError(s.db.QueryRowContext(ctx, `SELECT count(*) FROM account_pardons WHERE account = $1`, account).Scan(&accountPardons))
	s.Equal(2, pardons)
	s.Equal(2, accountPardons)
}
