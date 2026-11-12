package postgres_contribution_store_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	standings.StoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "scoring", migrations.FS)
	store := postgres_contribution_store.New(s.db)
	s.NewStore = func() standings.Store { return store }
	s.TallyOf = func(_ standings.Store, book books.BookID, account standings.AccountID) standings.Tally {
		return s.tallyOf(book, account)
	}
}

func (s *testSuite) tallyOf(book books.BookID, account standings.AccountID) standings.Tally {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(),
		`SELECT country, tiles, main, counted_takes_below, counted_takes_below_index FROM contributions
		WHERE book_id = $1 AND account_id = $2 ORDER BY counted_takes_below, counted_takes_below_index`,
		string(book), uuid.UUID(account))
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	tally := standings.Tally{Tiles: map[standings.Country]uint64{}}
	for rows.Next() {
		var (
			country string
			tiles   uint64
			main    bool
			below   uint64
			index   uint32
		)
		s.Require().NoError(rows.Scan(&country, &tiles, &main, &below, &index))
		tally.Tiles[standings.Country(country)] = tiles
		tally.TakesBelow = standings.TakeMark{Position: below, Index: index}
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

func (s *testSuite) TestOneRowIsMainPerAccountAndBook() {
	ada, bob := standings.AccountID{15: 1}, standings.AccountID{15: 2}
	store := postgres_contribution_store.New(s.db)
	countries := []standings.Country{"fr", "de", "de", "it", "it", "it"}
	batch := make([]standings.Take, 0, len(countries))
	for position, country := range countries {
		take := standings.Take{Country: country, Position: uint64(position)}
		take.Account = ada
		s.Require().NoError(store.RecordTakes(s.T().Context(), "season-0", ada, []standings.Take{take}))
		take.Account = bob
		batch = append(batch, take)
	}
	s.Require().NoError(store.RecordTakes(s.T().Context(), "season-0", bob, batch))

	var mains, rows int
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(),
		`SELECT count(*) FILTER (WHERE main), count(*) FROM contributions WHERE account_id = $1`, uuid.UUID(ada)).Scan(&mains, &rows))
	s.Equal(1, mains, "a take at a time")
	s.Equal(3, rows)
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(),
		`SELECT count(*) FILTER (WHERE main), count(*) FROM contributions WHERE account_id = $1`, uuid.UUID(bob)).Scan(&mains, &rows))
	s.Equal(1, mains, "the takes in one batch")
	s.Equal(3, rows)
}
