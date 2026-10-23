package names_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/find_authors_handler/names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db      *cppg.Postgres
	players *postgres_player_store.Store
}

var (
	ada     = players.AccountID{15: 1}
	eire    = players.AccountID{15: 2}
	eire1   = players.AccountID{15: 3}
	iraEire = players.AccountID{15: 4}
	guest   = players.AccountID{15: 5}
	today   = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.named(ada, "Ada")
	s.named(iraEire, "IRA EIRE")
	s.named(eire1, "EIRE1")
	s.named(eire, "EIRE")
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), guest, "a1b2c3"))
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), ada, "91aa3d"))
}

func (s *testSuite) named(account players.AccountID, name players.Name) {
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(account, name, today)))
}

func (s *testSuite) find(folded string, limit int) ([]players.AccountID, int) {
	accounts, total, err := names_query.NewPostgresQuery(s.db).Find(s.T().Context(), folded, limit)
	s.Require().NoError(err)
	return accounts, total
}

func (s *testSuite) TestAPartOfANameFindsTheSameNameFirstThenTheNamesItStarts() {
	accounts, total := s.find("eire", 20)
	s.Equal([]players.AccountID{eire, eire1, iraEire}, accounts)
	s.Equal(3, total)
}

func (s *testSuite) TestTheLimitCutsAndTheTotalCountsEveryAccountFound() {
	accounts, total := s.find("eire", 1)
	s.Equal([]players.AccountID{eire}, accounts)
	s.Equal(3, total)
}

func (s *testSuite) TestAGuestIsFoundByItsCodeUntilItChoosesAUsername() {
	accounts, _ := s.find("guest_a1", 20)
	s.Equal([]players.AccountID{guest}, accounts)

	accounts, total := s.find("91aa", 20)
	s.Empty(accounts, "the chat and the back office name Ada by its username, not the code it went by")
	s.Zero(total)
}

func (s *testSuite) TestADeletedAccountIsNotFound() {
	s.Require().NoError(s.players.DeleteAccount(s.T().Context(), eire))
	accounts, _ := s.find("eire", 20)
	s.Equal([]players.AccountID{eire1, iraEire}, accounts)
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()
	_, _, err := names_query.NewPostgresQuery(s.db).Find(ctx, "eire", 20)
	s.Error(err)
}
