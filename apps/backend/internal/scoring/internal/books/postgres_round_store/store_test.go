package postgres_round_store_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheRoundsAsTheContractSays(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)

	suite.Run(t, &books.RoundsContractSuite{NewStores: func() (books.Books, books.Rounds) {
		require.NoError(t, db.Purge(t.Context()))
		return postgres_book_store.New(db), postgres_round_store.New(db)
	}})
}

func TestThePostgresStoreCountsTheTakesAsTheContractSays(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)
	store := postgres_round_store.New(db)

	suite.Run(t, &books.TakesContractSuite{
		NewStores: func() (books.Books, books.Rounds, books.Takes) {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_book_store.New(db), store, store
		},
		CountedIn: func(round books.RoundID) books.Counted { return countedIn(t, db, round) },
	})
}

func countedIn(t *testing.T, db *cppg.Postgres, round books.RoundID) books.Counted {
	t.Helper()

	counted := books.Counted{Countries: map[books.Country]uint64{}, Players: map[books.AccountID]map[books.Country]uint64{}}
	countries, err := db.On(t.Context()).QueryContext(t.Context(),
		`SELECT country, tiles FROM round_takes WHERE round_id = $1`, string(round))
	require.NoError(t, err)
	defer func() { _ = countries.Close() }()
	for countries.Next() {
		var (
			country string
			tiles   int64
		)
		require.NoError(t, countries.Scan(&country, &tiles))
		counted.Countries[books.Country(country)] = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
	}
	require.NoError(t, countries.Err())

	players, err := db.On(t.Context()).QueryContext(t.Context(),
		`SELECT account_id, country, tiles FROM round_players WHERE round_id = $1`, string(round))
	require.NoError(t, err)
	defer func() { _ = players.Close() }()
	for players.Next() {
		var (
			account uuid.UUID
			country string
			tiles   int64
		)
		require.NoError(t, players.Scan(&account, &country, &tiles))
		if counted.Players[books.AccountID(account)] == nil {
			counted.Players[books.AccountID(account)] = map[books.Country]uint64{}
		}
		counted.Players[books.AccountID(account)][books.Country(country)] = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
	}
	require.NoError(t, players.Err())
	return counted
}
