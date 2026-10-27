package migrations_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestARoundClosedOnItsAverageKeepsItsResultsWithItsAverageShareAndTheRoundInProgressStaysOpen(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", cppg.MigrationsBefore(t, migrations.FS, "20261027180000"))
	closed := time.Date(2026, 10, 26, 21, 0, 0, 0, time.UTC)
	open := closed.Add(rounds.Length)
	for _, statement := range []string{
		`INSERT INTO rounds (season, ends_at, finale, samples, map_tiles, closed) VALUES
			(0, '2026-10-26T21:00:00Z', false, 2, 100, true), (0, '2026-10-27T21:00:00Z', false, 1, 100, false)`,
		`INSERT INTO round_holdings (season, ends_at, country, tiles) VALUES
			(0, '2026-10-26T21:00:00Z', 'fr', 50), (0, '2026-10-26T21:00:00Z', 'de', 20), (0, '2026-10-27T21:00:00Z', 'fr', 9)`,
		`INSERT INTO round_results (season, ends_at, country, rank, points) VALUES
			(0, '2026-10-26T21:00:00Z', 'fr', 1, 25), (0, '2026-10-26T21:00:00Z', 'de', 2, 18)`,
	} {
		_, err := db.On(t.Context()).ExecContext(t.Context(), statement)
		require.NoError(t, err)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	store := postgres_round_store.New(db)
	kept, err := store.ClosedIn(t.Context(), 0)
	require.NoError(t, err)
	assert.Equal(t, []rounds.Closed{{Round: rounds.Round{EndsAt: closed, Times: 1}, Number: 1, Results: []rounds.Result{
		{Country: "fr", Rank: 1, Points: 25, Share: 0.25},
		{Country: "de", Rank: 2, Points: 18, Share: 0.1},
	}}}, kept)
	unclosed, err := store.Unclosed(t.Context(), open)
	require.NoError(t, err)
	assert.Equal(t, []rounds.Round{{EndsAt: open, Times: 1}}, unclosed, "the round in progress is closed at its end, from the map")
}

func TestAnEventLeftInTheOutboxIsRelayedUnderTheScoringName(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", cppg.MigrationsBefore(t, migrations.FS, "20261028090000"))
	_, err := db.On(t.Context()).ExecContext(t.Context(), `INSERT INTO outbox (event, payload) VALUES ('seasons.v1.RoundClosed', '')`)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	var event string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `SELECT event::text FROM outbox`).Scan(&event))
	assert.Equal(t, "scoring.v1.RoundClosed", event)
}
