package migrations_test

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func before(t *testing.T, version string) fs.FS {
	t.Helper()

	names, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)

	older := fstest.MapFS{}
	for _, name := range names {
		if name < version {
			data, err := fs.ReadFile(migrations.FS, name)
			require.NoError(t, err)
			older[name] = &fstest.MapFile{Data: data}
		}
	}
	return older
}

func TestUnicodeUsernamesCutTheLongNamesAndDeleteTheOnesACutWouldTake(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "player", before(t, "20260918190000"))
	at := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	for i, row := range []struct {
		name string
		age  time.Duration
	}{
		{"Ada", 0},
		{"Ada_Lovelace_18", 0},
		{"ada_lovelace_1815", time.Hour},
		{"Bob_the_builder_1", 2 * time.Hour},
		{"BOB_THE_BUILDER_2", time.Hour},
		{"Carol_Carolyn_Carr", 0},
	} {
		_, err := db.ExecContext(t.Context(),
			`INSERT INTO profiles (account_id, name, updated_at) VALUES ($1, $2, $3)`,
			fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1), row.name, at.Add(-row.age))
		require.NoError(t, err)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	rows, err := db.QueryContext(t.Context(), `SELECT name, name_folded FROM profiles ORDER BY name`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var kept []string
	for rows.Next() {
		var name, folded string
		require.NoError(t, rows.Scan(&name, &folded))
		assert.Equal(t, strings.ToLower(name), folded)
		kept = append(kept, name)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"Ada", "Ada_Lovelace_18", "Bob_the_builder", "Carol_Carolyn_C"}, kept)
}

func TestTitlesAreBackfilledFromTheStats(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "player", before(t, "20261002120000"))
	for _, row := range []struct {
		account       string
		tiles, streak int
	}{
		{"00000000-0000-0000-0000-000000000001", 99, 6},
		{"00000000-0000-0000-0000-000000000002", 100, 7},
		{"00000000-0000-0000-0000-000000000003", 10_000, 30},
		{"00000000-0000-0000-0000-000000000004", 100_000, 100},
	} {
		_, err := db.ExecContext(t.Context(), `
			INSERT INTO stats (account_id, tiles_taken, streak_current, streak_best, streak_last_day)
			VALUES ($1, $2, 0, $3, '2026-10-01')
		`, row.account, row.tiles, row.streak)
		require.NoError(t, err)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	rows, err := db.QueryContext(t.Context(), `SELECT account_id::text, title FROM titles`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	held := map[string][]string{}
	for rows.Next() {
		var account, title string
		require.NoError(t, rows.Scan(&account, &title))
		held[account] = append(held[account], title)
	}
	require.NoError(t, rows.Err())
	assert.Empty(t, held["00000000-0000-0000-0000-000000000001"])
	assert.ElementsMatch(t, []string{"settler", "loyal"}, held["00000000-0000-0000-0000-000000000002"])
	assert.ElementsMatch(t, []string{"settler", "governor", "conqueror", "loyal", "devoted"},
		held["00000000-0000-0000-0000-000000000003"])
	assert.ElementsMatch(t, []string{"settler", "governor", "conqueror", "emperor", "loyal", "devoted", "unbroken"},
		held["00000000-0000-0000-0000-000000000004"])
}
