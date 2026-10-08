package postgres_front_store_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/postgres_front_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	fronts.StoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	store := postgres_front_store.New(s.db)
	s.NewStore = func() fronts.Store { return store }
	s.TallyOf = func(_ fronts.Store, account players.AccountID) fronts.Tally {
		return s.tallyOf(account)
	}
}

func (s *testSuite) tallyOf(account players.AccountID) fronts.Tally {
	rows, err := s.db.QueryContext(s.T().Context(),
		`SELECT country, plays_for, plays_against FROM fronts WHERE account_id = $1`, uuid.UUID(account))
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	playsFor, playsAgainst := map[fronts.Country]uint64{}, map[fronts.Country]uint64{}
	for rows.Next() {
		var (
			country      string
			tilesFor     uint64
			tilesAgainst uint64
		)
		s.Require().NoError(rows.Scan(&country, &tilesFor, &tilesAgainst))
		playsFor[fronts.Country(country)] = tilesFor
		playsAgainst[fronts.Country(country)] = tilesAgainst
	}
	s.Require().NoError(rows.Err())
	return fronts.TallyOf(playsFor, playsAgainst)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

func (s *testSuite) TestACountryIsOneRowForAndAgainst() {
	for _, pair := range [][2]fronts.Country{{"fr", "de"}, {"de", "fr"}, {"fr", ""}} {
		take, err := fronts.NewTake(players.AccountID{15: 1}, pair[0], pair[1])
		s.Require().NoError(err)
		s.Require().NoError(postgres_front_store.New(s.db).RecordTake(s.T().Context(), take))
	}

	var rows int
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT count(*) FROM fronts`).Scan(&rows))
	s.Equal(2, rows)
}
