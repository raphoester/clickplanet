package migrations_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

func seed(t *testing.T, db *cppg.Postgres, statements ...string) {
	t.Helper()
	for _, statement := range statements {
		_, err := db.On(t.Context()).ExecContext(t.Context(), statement)
		require.NoError(t, err, statement)
	}
}

func TestARoundClosedOnItsAverageKeepsItsResultsWithItsAverageShareAndTheRoundInProgressStaysOpen(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", cppg.MigrationsBefore(t, migrations.FS, "20261027180000"))
	seed(t, db,
		`INSERT INTO rounds (season, ends_at, finale, samples, map_tiles, closed) VALUES
			(0, '2026-10-26T21:00:00Z', false, 2, 100, true), (0, '2026-10-27T21:00:00Z', false, 1, 100, false)`,
		`INSERT INTO round_holdings (season, ends_at, country, tiles) VALUES
			(0, '2026-10-26T21:00:00Z', 'fr', 50), (0, '2026-10-26T21:00:00Z', 'de', 20), (0, '2026-10-27T21:00:00Z', 'fr', 9)`,
		`INSERT INTO round_results (season, ends_at, country, rank, points) VALUES
			(0, '2026-10-26T21:00:00Z', 'fr', 1, 25), (0, '2026-10-26T21:00:00Z', 'de', 2, 18)`,
	)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	store := postgres_round_store.New(db)
	kept, err := store.ScoredIn(t.Context(), "season-0")
	require.NoError(t, err)
	assert.Equal(t, []books.Scored{{
		Round: books.Round{ID: "season-0/day-19", Book: "season-0", Times: 1, StartsAt: utc(25, 21), Scored: true, EndsAt: utc(26, 21)},
		Results: []books.Result{
			{Country: "fr", Rank: 1, Points: 25, Share: 0.25},
			{Country: "de", Rank: 2, Points: 18, Share: 0.1},
		},
	}}, kept)
	open, found, err := store.Find(t.Context(), "season-0/day-20")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, books.Round{ID: "season-0/day-20", Book: "season-0", Times: 1, StartsAt: utc(26, 21)}, open,
		"the round in progress is scored at its end, from the map")
}

func TestAnEventOfAKindNoLongerSentIsDroppedAndTheOutboxKeepsTheNewKinds(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", cppg.MigrationsBefore(t, migrations.FS, "20261028090000"))
	seed(t, db, `INSERT INTO outbox (event, payload) VALUES ('seasons.v1.RoundClosed', ''), ('seasons.v1.SeasonClosed', '')`)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	var left int
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `SELECT count(*) FROM outbox`).Scan(&left))
	assert.Zero(t, left)
	seed(t, db, `INSERT INTO outbox (event, payload) VALUES ('scoring.v1.RoundScored', ''), ('scoring.v1.BookClosed', '')`)
}

const battle = "0192f5a0-7c00-7000-8000-000000000031"

func TestSeasonZeroBecomesABookAndItsRoundsKeepTheirDayNumbersAndTheirResults(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", cppg.MigrationsBefore(t, migrations.FS, "20261028130000"))
	account := uuid.MustParse("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	seed(t, db,
		`INSERT INTO rounds (season, ends_at, own, finale, starts_at, times, closed, counted_takes_below) VALUES
			(0, '2026-10-26T21:00:00Z', false, false, NULL, 1, true, 0),
			(0, '2026-10-27T21:00:00Z', false, false, NULL, 1, true, 0),
			(0, '2026-10-28T21:00:00Z', false, false, NULL, 1, false, 0),
			(0, '2026-10-27T20:00:00Z', true, false, '2026-10-27T18:00:00Z', 2, true, 41),
			(0, '2026-10-28T20:00:00Z', true, false, '2026-10-28T18:00:00Z', 3, false, 0)`,
		`INSERT INTO round_results (season, ends_at, own, country, rank, points, share) VALUES
			(0, '2026-10-26T21:00:00Z', false, 'fr', 1, 25, 0.3), (0, '2026-10-26T21:00:00Z', false, 'de', 2, 18, 0.2),
			(0, '2026-10-27T21:00:00Z', false, 'de', 1, 25, 0.4),
			(0, '2026-10-27T20:00:00Z', true, 'it', 1, 50, 0.1)`,
		`INSERT INTO own_rounds (season, ends_at, own, entry_id) VALUES (0, '2026-10-27T20:00:00Z', true, '`+battle+`')`,
		`INSERT INTO round_takes (season, ends_at, own, country, tiles) VALUES (0, '2026-10-27T20:00:00Z', true, 'it', 7)`,
		`INSERT INTO round_players (season, ends_at, own, account_id, country, tiles) VALUES
			(0, '2026-10-27T20:00:00Z', true, '`+account.String()+`', 'it', 7)`,
		`INSERT INTO contributions (season, account_id, country, tiles, main, counted_takes_below) VALUES
			(0, '`+account.String()+`', 'it', 7, true, 41), (0, '`+account.String()+`', 'fr', 2, false, 41)`,
		`INSERT INTO rule_set_choice (rule_set, decided_at) VALUES ('default', '2026-10-27T00:00:00Z')`,
	)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	book, found, err := postgres_book_store.New(db).Find(t.Context(), "season-0")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, books.NewBook("season-0", utc(8, 0)), book, "open, from the gamemaster's start of season 0")

	store := postgres_round_store.New(db)
	scored, err := store.ScoredIn(t.Context(), "season-0")
	require.NoError(t, err)
	assert.Equal(t, []books.Scored{
		{
			Round: books.Round{ID: "season-0/day-19", Book: "season-0", Times: 1, StartsAt: utc(25, 21), Scored: true, EndsAt: utc(26, 21)},
			Results: []books.Result{
				{Country: "fr", Rank: 1, Points: 25, Share: 0.3}, {Country: "de", Rank: 2, Points: 18, Share: 0.2},
			},
		},
		{
			Round:   books.Round{ID: battle, Book: "season-0", Times: 2, StartsAt: utc(27, 18), Scored: true, EndsAt: utc(27, 20)},
			Results: []books.Result{{Country: "it", Rank: 1, Points: 50, Share: 0.1}},
		},
		{
			Round:   books.Round{ID: "season-0/day-20", Book: "season-0", Times: 1, StartsAt: utc(26, 21), Scored: true, EndsAt: utc(27, 21)},
			Results: []books.Result{{Country: "de", Rank: 1, Points: 25, Share: 0.4}},
		},
	}, scored)

	open, err := store.Of(t.Context(), []books.BookID{"season-0"})
	require.NoError(t, err)
	assert.Contains(t, open, books.Round{ID: "season-0/day-21", Book: "season-0", Times: 1, StartsAt: utc(27, 21)},
		"the day in progress is still open, and the gamemaster's CloseRound at 21:00 scores it")
	assert.Contains(t, open, books.Round{ID: "season-0/own-2", Book: "season-0", Times: 3, StartsAt: utc(28, 18)},
		"a round of its own with no entry is named by its place")

	var counted, takes, players, contributions, mains int
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `
		SELECT (SELECT counted_takes_below FROM rounds WHERE round_id = $1),
		       (SELECT tiles FROM round_takes WHERE round_id = $1 AND country = 'it'),
		       (SELECT tiles FROM round_players WHERE round_id = $1 AND account_id = $2),
		       (SELECT count(*) FROM contributions WHERE book_id = 'season-0' AND account_id = $2),
		       (SELECT tiles FROM contributions WHERE book_id = 'season-0' AND account_id = $2 AND main)
	`, battle, account).Scan(&counted, &takes, &players, &contributions, &mains))
	assert.Equal(t, []int{41, 7, 7, 2, 7}, []int{counted, takes, players, contributions, mains})

	for _, gone := range []string{"rule_sets", "rule_set_choice", "own_rounds", "ended_seasons"} {
		var exists bool
		require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
			`SELECT to_regclass($1) IS NOT NULL`, gone).Scan(&exists))
		assert.False(t, exists, gone)
	}
}

func TestASeasonEndedIsABookClosedWithItsFinaleAsTheTiebreak(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", cppg.MigrationsBefore(t, migrations.FS, "20261028130000"))
	seed(t, db,
		`INSERT INTO rounds (season, ends_at, own, finale, starts_at, times, closed) VALUES
			(0, '2026-10-31T21:00:00Z', false, false, NULL, 1, true),
			(0, '2026-10-31T23:00:00Z', true, true, '2026-10-31T21:00:00Z', 3, true)`,
		`INSERT INTO own_rounds (season, ends_at, own, entry_id) VALUES (0, '2026-10-31T23:00:00Z', true, '`+battle+`')`,
		`INSERT INTO ended_seasons (season, ended_at) VALUES (0, '2026-10-31T23:00:00Z')`,
	)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	book, _, err := postgres_book_store.New(db).Find(t.Context(), "season-0")
	require.NoError(t, err)
	assert.Equal(t, books.NewBook("season-0", utc(8, 0)).ClosedAt(utc(31, 23), battle), book)
	day, _, err := postgres_round_store.New(db).Find(t.Context(), "season-0/day-24")
	require.NoError(t, err)
	assert.True(t, day.Scored)
}

func TestAnotherSeasonThanZeroRefusesTheConversion(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", cppg.MigrationsBefore(t, migrations.FS, "20261028130000"))
	seed(t, db, `INSERT INTO contributions (season, account_id, country, tiles, main) VALUES
		(1, '0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11', 'fr', 1, true)`)

	require.Error(t, db.Migrate(t.Context(), migrations.FS))
}
