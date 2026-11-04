package migrations_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Player's tables as its own migrations left them, with a row each, and what the move reads of its profiles and
// departures.
const playerAsItWas = `
CREATE TABLE stats (
    account_id uuid PRIMARY KEY, tiles_taken bigint NOT NULL, streak_current bigint NOT NULL, streak_best bigint NOT NULL,
    streak_last_day date
);
CREATE TABLE titles (account_id uuid NOT NULL, title text NOT NULL, earned_at timestamptz NOT NULL, PRIMARY KEY (account_id, title));
CREATE TABLE worn_titles (account_id uuid PRIMARY KEY, title text NOT NULL, worn_at timestamptz NOT NULL);
CREATE TABLE finale_fights (
    season integer NOT NULL, account_id uuid NOT NULL, country text NOT NULL, flag_rank integer NOT NULL,
    tiles_rank integer NOT NULL DEFAULT 0, PRIMARY KEY (season, account_id)
);
CREATE INDEX finale_fights_account ON finale_fights (account_id);
CREATE TABLE profiles (account_id uuid PRIMARY KEY);
CREATE TABLE departures (account_id uuid PRIMARY KEY, leaving boolean NOT NULL, decided_at timestamptz NOT NULL);

INSERT INTO stats VALUES ('00000000-0000-0000-0000-000000000001', 12, 2, 3, '2026-11-03');
INSERT INTO titles VALUES ('00000000-0000-0000-0000-000000000001', 'og', '2026-10-01T12:00:00Z');
INSERT INTO worn_titles VALUES ('00000000-0000-0000-0000-000000000001', 'og', '2026-10-01T12:00:00Z');
INSERT INTO finale_fights VALUES (0, '00000000-0000-0000-0000-000000000001', 'fr', 1, 2);
INSERT INTO profiles VALUES ('00000000-0000-0000-0000-000000000001'), ('00000000-0000-0000-0000-000000000002');
INSERT INTO departures VALUES ('00000000-0000-0000-0000-000000000002', true, '2026-11-02T12:00:00Z');
`

func TestTheTablesMoveFromPlayerWithTheirRows(t *testing.T) {
	server := cppg.StartTestServer(t)
	player := server.OpenSchema(t, "player", fstest.MapFS{"1_player.up.sql": {Data: []byte(playerAsItWas)}})
	progression := server.OpenSchema(t, "progression", migrations.FS)
	ctx := t.Context()

	var tiles, titles, worn, fights int
	require.NoError(t, progression.On(ctx).QueryRowContext(ctx, `SELECT
		(SELECT tiles_taken FROM stats),
		(SELECT count(*) FROM titles),
		(SELECT count(*) FROM worn_titles),
		(SELECT tiles_rank FROM finale_fights)`).Scan(&tiles, &titles, &worn, &fights))
	assert.Equal(t, []int{12, 1, 1, 2}, []int{tiles, titles, worn, fights})

	var left int
	require.NoError(t, player.On(ctx).QueryRowContext(ctx, `SELECT count(*) FROM pg_tables
		WHERE schemaname = 'player' AND tablename IN ('stats', 'titles', 'worn_titles', 'finale_fights')`).Scan(&left))
	assert.Zero(t, left, "moved, not copied")

	var index bool
	require.NoError(t, progression.On(ctx).QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'progression' AND indexname = 'finale_fights_account')`).Scan(&index))
	assert.True(t, index, "the index moves with its table")
}

func TestTheNameplatesAndTheDeparturesStartFromPlayers(t *testing.T) {
	server := cppg.StartTestServer(t)
	server.OpenSchema(t, "player", fstest.MapFS{"1_player.up.sql": {Data: []byte(playerAsItWas)}})
	progression := server.OpenSchema(t, "progression", migrations.FS)
	ctx := t.Context()

	var named, leaving int
	require.NoError(t, progression.On(ctx).QueryRowContext(ctx, `SELECT
		(SELECT count(*) FROM nameplates),
		(SELECT count(*) FROM departures WHERE leaving AND decided_at = '2026-11-02T12:00:00Z')`).Scan(&named, &leaving))
	assert.Equal(t, 2, named)
	assert.Equal(t, 1, leaving)
}

func TestAFreshDatabaseMakesTheTables(t *testing.T) {
	progression := cppg.StartTestServer(t).OpenSchema(t, "progression", migrations.FS)
	ctx := t.Context()

	var tables int
	require.NoError(t, progression.On(ctx).QueryRowContext(ctx, `SELECT count(*) FROM pg_tables WHERE schemaname = 'progression'
		AND tablename IN ('stats', 'titles', 'worn_titles', 'finale_fights', 'nameplates', 'departures')`).Scan(&tables))
	assert.Equal(t, 6, tables)
}
