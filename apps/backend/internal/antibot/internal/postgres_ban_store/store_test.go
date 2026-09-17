package postgres_ban_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_ban_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db       *cppg.Postgres
	store    *postgres_ban_store.Store
	accounts *postgres_ban_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "antibot", migrations.FS)
	s.store = postgres_ban_store.NewScopes(s.db)
	s.accounts = postgres_ban_store.NewAccounts(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) load() map[string]shadowban.Record {
	return s.loadFrom(s.store)
}

func (s *testSuite) loadFrom(store *postgres_ban_store.Store) map[string]shadowban.Record {
	loaded := map[string]shadowban.Record{}
	s.Require().NoError(store.Load(s.T().Context(), func(record shadowban.Record) {
		loaded[record.Key] = record
	}))
	return loaded
}

// until is whole microseconds, the precision postgres keeps.
var until = time.Date(2026, 9, 16, 12, 30, 0, 123456000, time.UTC)

func (s *testSuite) TestAnEmptyTableLoadsNothing() {
	s.Empty(s.load())
}

func (s *testSuite) TestSaveThenLoad() {
	s.Require().NoError(s.store.Save(s.T().Context(), []shadowban.Record{
		{Key: "1.2.3.4", Flags: 3, Offences: 2, Until: until},
		{Key: "2a00:8c40:f0c5:6713::/64", Flags: 1, Offences: 1, Until: until.Add(time.Hour)},
	}))

	loaded := s.load()
	s.Require().Len(loaded, 2)
	s.Equal(3, loaded["1.2.3.4"].Flags)
	s.Equal(2, loaded["1.2.3.4"].Offences)
	s.True(loaded["1.2.3.4"].Until.Equal(until))
	s.True(loaded["2a00:8c40:f0c5:6713::/64"].Until.Equal(until.Add(time.Hour)))
}

func (s *testSuite) TestSaveOverwritesAScopeAndKeepsTheOthers() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Save(ctx, []shadowban.Record{
		{Key: "kept", Flags: 1, Offences: 1, Until: until},
		{Key: "changed", Flags: 1, Offences: 1, Until: until},
	}))

	s.Require().NoError(s.store.Save(ctx, []shadowban.Record{{Key: "changed", Flags: 2, Offences: 2, Until: until.Add(24 * time.Hour)}}))

	loaded := s.load()
	s.Require().Len(loaded, 2)
	s.Equal(1, loaded["kept"].Flags)
	s.Equal(2, loaded["changed"].Offences)
	s.True(loaded["changed"].Until.Equal(until.Add(24 * time.Hour)))
}

func (s *testSuite) TestAFailedSaveWritesNothing() {
	s.Require().Error(s.store.Save(s.T().Context(), []shadowban.Record{
		{Key: "fine", Flags: 1, Offences: 1, Until: until},
		{Key: "", Flags: 1, Offences: 1, Until: until},
	}))

	s.Empty(s.load())
}

func (s *testSuite) TestAccountsAreKeptApartFromScopes() {
	ctx := s.T().Context()
	const account = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

	s.Require().NoError(s.accounts.Save(ctx, []shadowban.Record{{Key: account, Flags: 2, Offences: 1, Until: until}}))
	s.Require().NoError(s.store.Save(ctx, []shadowban.Record{{Key: "1.2.3.4", Flags: 1, Offences: 1, Until: until}}))

	loaded := s.loadFrom(s.accounts)
	s.Require().Len(loaded, 1)
	s.Equal(2, loaded[account].Flags)
	s.True(loaded[account].Until.Equal(until))
	s.NotContains(s.load(), account)
}

func (s *testSuite) TestAKeyThatIsNotAnAccountWritesNothing() {
	s.Require().Error(s.accounts.Save(s.T().Context(), []shadowban.Record{
		{Key: "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", Flags: 1, Offences: 1, Until: until},
		{Key: "1.2.3.4", Flags: 1, Offences: 1, Until: until},
	}))

	s.Empty(s.loadFrom(s.accounts))
}
