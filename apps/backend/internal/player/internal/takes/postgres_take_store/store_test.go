package postgres_take_store_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/postgres_take_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	takes.StoreContractSuite

	db      *cppg.Postgres
	store   *postgres_take_store.Store
	players *postgres_player_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.store = postgres_take_store.New(s.db)
	s.players = postgres_player_store.New(s.db)
	s.NewStore = func() (takes.Store, takes.Players) { return s.store, s.players }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

var (
	ada = players.AccountID{15: 1}
	at  = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
)

func (s *testSuite) batch(from takes.Position, list ...takes.Take) takes.Batch {
	batch, err := takes.BatchOf(from, list[len(list)-1].Position()+1, list)
	s.Require().NoError(err)
	return batch
}

func (s *testSuite) tiles(account players.AccountID) uint64 {
	stats, err := s.players.Stats(s.T().Context(), account)
	s.Require().NoError(err)
	return stats.TilesTaken()
}

func (s *testSuite) TestACrashBetweenTheCountAndTheSaveOfThePositionCountsNothingAndTheRetryCountsOnce() {
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))
	_, err := s.db.ExecContext(s.T().Context(), `
		CREATE FUNCTION crash() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'crash'; END $$;
		CREATE TRIGGER crash BEFORE UPDATE ON stats_position FOR EACH ROW EXECUTE FUNCTION crash();
	`)
	s.Require().NoError(err)
	heal := func() {
		_, err := s.db.ExecContext(s.T().Context(), `DROP TRIGGER IF EXISTS crash ON stats_position; DROP FUNCTION IF EXISTS crash();`)
		s.Require().NoError(err)
	}
	s.T().Cleanup(heal)
	batch := s.batch(0, takes.TakeOf(0, ada, "fr", at, false), takes.TakeOf(1, ada, "fr", at, false))

	s.Require().Error(s.store.Count(s.T().Context(), batch))

	_, err = s.players.Stats(s.T().Context(), ada)
	s.Require().ErrorIs(err, players.ErrNoStats, "the stats written before the crash are gone with it")
	position, err := s.store.Position(s.T().Context())
	s.Require().NoError(err)
	s.Equal(takes.Position(0), position)

	heal()
	s.Require().NoError(s.store.Count(s.T().Context(), batch))

	s.Equal(uint64(2), s.tiles(ada))
}

func (s *testSuite) TestTheStreakDayIsStoredAsADate() {
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))

	s.Require().NoError(s.store.Count(s.T().Context(), s.batch(0, takes.TakeOf(0, ada, "fr", at, false))))

	var day string
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT streak_last_day::text FROM stats`).Scan(&day))
	s.Equal("2026-10-05", day)
}

func (s *testSuite) TestMessagesCountedWhileBatchesAreCountedAreAllKept() {
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			<-start
			s.NoError(s.players.RecordMessage(s.T().Context(), ada))
		})
	}
	wg.Go(func() {
		<-start
		for position := range takes.Position(20) {
			s.NoError(s.store.Count(s.T().Context(), s.batch(position, takes.TakeOf(position, ada, "fr", at, false))))
		}
	})
	close(start)
	wg.Wait()

	stats, err := s.players.Stats(s.T().Context(), ada)
	s.Require().NoError(err)
	s.Equal(uint64(20), stats.MessagesSent())
	s.Equal(uint64(20), stats.TilesTaken())
}

func (s *testSuite) TestTheBaselineIsATableOfItsOwnWithOnlyTheAccountsThatTookATile() {
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, at))
	s.Require().NoError(s.players.RecordMessage(s.T().Context(), players.AccountID{15: 2}))

	s.Require().NoError(s.store.Begin(s.T().Context(), 7))

	var accounts int
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT count(*) FROM stats_baseline`).Scan(&accounts))
	s.Equal(1, accounts)
}
