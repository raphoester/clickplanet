package postgres_contribution_store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	standings.StoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "seasons", migrations.FS)
	store := postgres_contribution_store.New(s.db)
	s.NewStore = func() standings.Store { return store }
	s.TallyOf = func(_ standings.Store, season calendar.Number, account standings.AccountID) standings.Tally {
		return s.tallyOf(season, account)
	}
}

func (s *testSuite) tallyOf(season calendar.Number, account standings.AccountID) standings.Tally {
	rows, err := s.db.QueryContext(s.T().Context(),
		`SELECT country, tiles, main FROM contributions WHERE season = $1 AND account_id = $2`, int64(season), uuid.UUID(account))
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	tally := standings.Tally{Tiles: map[standings.Country]uint64{}}
	for rows.Next() {
		var (
			country string
			tiles   uint64
			main    bool
		)
		s.Require().NoError(rows.Scan(&country, &tiles, &main))
		tally.Tiles[standings.Country(country)] = tiles
		if main {
			tally.Main = standings.Country(country)
		}
	}
	s.Require().NoError(rows.Err())
	return tally
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

func (s *testSuite) TestOneRowIsMainPerAccountAndSeason() {
	for _, country := range []standings.Country{"fr", "de", "de", "it", "it", "it"} {
		s.Require().NoError(postgres_contribution_store.New(s.db).RecordTake(s.T().Context(), 0,
			standings.Take{Account: standings.AccountID{15: 1}, Country: country}))
	}

	var mains, rows int
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(),
		`SELECT count(*) FILTER (WHERE main), count(*) FROM contributions`).Scan(&mains, &rows))
	s.Equal(1, mains)
	s.Equal(3, rows)
}

func (s *testSuite) TestACrashBetweenTheCountAndTheSaveOfThePositionCountsNothingAndTheRetryCountsOnce() {
	store := postgres_contribution_store.New(s.db)
	s.Require().NoError(store.Begin(s.T().Context(), 0))
	_, err := s.db.ExecContext(s.T().Context(), `
		CREATE FUNCTION crash() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'crash'; END $$;
		CREATE TRIGGER crash BEFORE UPDATE ON standings_position FOR EACH ROW EXECUTE FUNCTION crash();
	`)
	s.Require().NoError(err)
	heal := func() {
		_, err := s.db.ExecContext(s.T().Context(), `DROP TRIGGER IF EXISTS crash ON standings_position; DROP FUNCTION IF EXISTS crash();`)
		s.Require().NoError(err)
	}
	s.T().Cleanup(heal)
	ada := standings.AccountID{15: 1}
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	seasons := calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: at.Add(time.Hour), Finale: time.Minute}}})
	batch, err := standings.BatchOf(0, 2, []standings.Entry{
		standings.EntryOf(0, standings.Take{Account: ada, Country: "fr", At: at}, false),
		standings.EntryOf(1, standings.Take{Account: ada, Country: "fr", At: at}, false),
	})
	s.Require().NoError(err)

	s.Require().Error(store.Count(s.T().Context(), batch, seasons))
	s.True(s.tallyOf(0, ada).Empty(), "the tiles written before the crash are gone with it")

	heal()
	s.Require().NoError(store.Count(s.T().Context(), batch, seasons))
	s.Equal(map[standings.Country]uint64{"fr": 2}, s.tallyOf(0, ada).Tiles)
}
