package migrations_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestARoundClosedOnItsAverageKeepsItsResultsWithItsAverageShareAndTheRoundInProgressStaysOpen(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "seasons", cppg.MigrationsBefore(t, migrations.FS, "20261027180000"))
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
	assert.Equal(t, []rounds.Closed{{Round: rounds.Round{EndsAt: closed}, Number: 1, Results: []rounds.Result{
		{Country: "fr", Rank: 1, Points: 25, Share: 0.25},
		{Country: "de", Rank: 2, Points: 18, Share: 0.1},
	}}}, kept)
	unclosed, err := store.Unclosed(t.Context(), open)
	require.NoError(t, err)
	assert.Equal(t, []rounds.Round{{EndsAt: open}}, unclosed, "the round in progress is closed at its end, from the map")
}
