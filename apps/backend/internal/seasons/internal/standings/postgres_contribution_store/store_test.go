package postgres_contribution_store_test

import (
	"testing"

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
