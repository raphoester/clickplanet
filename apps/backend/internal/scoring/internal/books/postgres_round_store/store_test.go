package postgres_round_store_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

func TestACloseWaitsForTheOpenThatHoldsTheRoundAndFindsItOpen(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)
	ctx := t.Context()
	start := time.Date(2026, 11, 8, 21, 0, 0, 0, time.UTC)
	_, err := postgres_book_store.New(db).Open(ctx, books.NewBook("season-1", start))
	require.NoError(t, err)
	store := postgres_round_store.New(db)
	round := books.Round{ID: "season-1/day-1", Book: "season-1", Times: 1, Ranking: books.GainRanking{}, StartsAt: start}
	locked, release, opened := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		opened <- db.InTx(ctx, func(ctx context.Context) error {
			if err := store.Lock(ctx, round.ID); err != nil {
				return err //nolint:wrapcheck // the store says what failed.
			}
			close(locked)
			<-release
			_, err := store.Open(ctx, round, books.Start{})
			return err //nolint:wrapcheck // the store says what failed.
		})
	}()
	select {
	case <-locked:
	case err := <-opened:
		require.FailNow(t, "the open never held the round", err)
	}

	closing := make(chan bool, 1)
	go func() {
		found := false
		assert.NoError(t, db.InTx(ctx, func(ctx context.Context) error {
			if err := store.Lock(ctx, round.ID); err != nil {
				return err //nolint:wrapcheck // the store says what failed.
			}
			var err error
			_, found, err = store.Find(ctx, round.ID)
			return err //nolint:wrapcheck // the store says what failed.
		}))
		closing <- found
	}()

	assert.Never(t, func() bool { return len(closing) > 0 }, 300*time.Millisecond, 10*time.Millisecond, "the close waits for the open")
	close(release)
	require.NoError(t, <-opened)
	assert.True(t, <-closing, "once the open commits, the close finds the round open")
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
