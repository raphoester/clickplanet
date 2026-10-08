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

func TestFrontsStartFromWhatEverySeasonCountedForEachFlag(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "player", before(t, "20261008120000"))
	_, err := db.ExecContext(t.Context(), `
		CREATE SCHEMA seasons;
		CREATE TABLE seasons.contributions (season integer, account_id uuid, country text, tiles bigint, main boolean);
		INSERT INTO seasons.contributions VALUES
			(0, '00000000-0000-0000-0000-000000000001', 'fr', 30, true),
			(0, '00000000-0000-0000-0000-000000000001', 'de', 4, false),
			(1, '00000000-0000-0000-0000-000000000001', 'fr', 2, true),
			(0, '00000000-0000-0000-0000-000000000002', 'it', 7, true);
	`)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	rows, err := db.QueryContext(t.Context(),
		`SELECT account_id::text, country, plays_for, plays_against FROM fronts ORDER BY account_id, country`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var kept []string
	for rows.Next() {
		var (
			account, country       string
			playsFor, playsAgainst int
		)
		require.NoError(t, rows.Scan(&account, &country, &playsFor, &playsAgainst))
		kept = append(kept, fmt.Sprintf("%s %s %d %d", account[len(account)-1:], country, playsFor, playsAgainst))
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"1 de 4 0", "1 fr 32 0", "2 it 7 0"}, kept)
}
