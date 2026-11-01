package migrations_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestUnicodeUsernamesCutTheLongNamesAndDeleteTheOnesACutWouldTake(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "player", cppg.MigrationsBefore(t, migrations.FS, "20260918190000"))
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
		_, err := db.On(t.Context()).ExecContext(t.Context(),
			`INSERT INTO profiles (account_id, name, updated_at) VALUES ($1, $2, $3)`,
			fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1), row.name, at.Add(-row.age))
		require.NoError(t, err)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	rows, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT name, name_folded FROM profiles ORDER BY name`)
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

func TestTheBattleTopsAlreadyGivenAreRenamedAndKeepTheirHolders(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "player", cppg.MigrationsBefore(t, migrations.FS, "20261101180100"))
	at := time.Date(2026, 11, 1, 14, 30, 0, 0, time.UTC)
	for _, row := range []struct{ account, title string }{
		{"00000000-0000-0000-0000-000000000001", "season_0_top_1"},
		{"00000000-0000-0000-0000-000000000001", "season_0_gold"},
		{"00000000-0000-0000-0000-000000000002", "season_0_top_3"},
	} {
		_, err := db.On(t.Context()).ExecContext(t.Context(),
			`INSERT INTO titles (account_id, title, earned_at) VALUES ($1, $2, $3)`, row.account, row.title, at)
		require.NoError(t, err)
	}
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO worn_titles (account_id, title, worn_at) VALUES ($1, 'season_0_top_1', $2)`,
		"00000000-0000-0000-0000-000000000001", at)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	rows, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT title FROM titles ORDER BY account_id, title`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var held []string
	for rows.Next() {
		var title string
		require.NoError(t, rows.Scan(&title))
		held = append(held, title)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"season_0_battle_top_1", "season_0_gold", "season_0_battle_top_3"}, held)

	var worn string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `SELECT title FROM worn_titles`).Scan(&worn))
	assert.Equal(t, "season_0_battle_top_1", worn)
}
