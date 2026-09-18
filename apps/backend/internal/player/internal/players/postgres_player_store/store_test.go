package postgres_player_store_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	players.StoreContractSuite

	db    *cppg.Postgres
	store *postgres_player_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.store = postgres_player_store.New(s.db)
	s.NewStore = func() players.Store { return s.store }
	// The statement an operator runs.
	s.MakeAdmin = func(_ players.Store, account players.AccountID) {
		_, err := s.db.ExecContext(s.T().Context(), `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(account))
		s.Require().NoError(err)
	}
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

var at = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func (s *testSuite) TestTheTableRefusesANameThatBreaksTheRule() {
	for _, name := range []string{
		strings.Repeat("a", 16), "ab", " Ada", "Ada ", "Ada  L", "Ada-L", "Ada!", "Ada\u200bL", "\u202eAda", "Ada\u0085", "E\u0301mile",
		"GUEST_ada",
	} {
		err := s.store.SaveProfile(s.T().Context(),
			players.Profile{Account: players.AccountID{15: 1}, Name: players.Name(name), UpdatedAt: at})

		s.Require().Error(err, "%q", name)
		s.NotErrorIs(err, players.ErrNameTaken, "%q", name)
	}
}

func (s *testSuite) TestTheTableTakesTheNamesTheRuleTakes() {
	for i, value := range []string{"Ada Lovelace", "Émile Zola", "東京タワー", "محمد", strings.Repeat("é", 15)} {
		name, err := players.NameOf(value)
		s.Require().NoError(err)

		s.Require().NoError(s.store.SaveProfile(s.T().Context(),
			players.Profile{Account: players.AccountID{15: byte(i + 1)}, Name: name, UpdatedAt: at}), "%q", value)
	}
}

func (s *testSuite) TestTheStreakDayIsStoredAsADate() {
	s.Require().NoError(s.store.RecordTake(s.T().Context(), players.AccountID{15: 1}, at))

	var day string
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT streak_last_day::text FROM stats`).Scan(&day))
	s.Equal("2026-09-17", day)
}

func (s *testSuite) TestConcurrentFirstTakesAreAllCounted() {
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			<-start
			s.NoError(s.store.RecordTake(s.T().Context(), players.AccountID{15: 1}, at))
		})
	}
	close(start)
	wg.Wait()

	stats, err := s.store.Stats(s.T().Context(), players.AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(uint64(20), stats.TilesTaken, "no take overwrites another, the first ones included")
}
