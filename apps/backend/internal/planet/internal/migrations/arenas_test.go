package migrations_test

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const (
	arenas      = "20261110120000"
	afterArenas = "20261110120100"
	dropOutbox  = "20261111170000"
	tellArenas  = `INSERT INTO outbox (event, payload) VALUES ('planet.v1.ArenaOpened', ''), ('planet.v1.ArenaClosed', '')`
)

func countWhere(t *testing.T, db *cppg.Postgres, query string) int {
	t.Helper()
	var count int
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), query).Scan(&count))
	return count
}

func TestEveryTileAndLockKeptBeforeTheArenasIsTheWorlds(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, arenas))
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO tiles (id, country) VALUES (1, 'fr'), (2, 'de');
		INSERT INTO landmasses (asset, id, fortified_by) VALUES ('borders.bin', 4, 'fr');
	`)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	assert.Equal(t, 2, countWhere(t, db, `SELECT count(*) FROM tiles WHERE map = 'world'`))
	assert.Equal(t, 1, countWhere(t, db, `SELECT count(*) FROM landmasses WHERE map = 'world'`))
	_, err = db.On(t.Context()).ExecContext(t.Context(), `INSERT INTO tiles (map, id, country) VALUES ('battle', 1, 'it')`)
	require.NoError(t, err, "one tile id on two maps")
}

func TestTheArenasGoBackWithTheWorldsMapAlone(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, afterArenas))
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO tiles (map, id, country) VALUES ('world', 1, 'fr'), ('battle', 1, 'it'), ('battle', 3, 'it');
		INSERT INTO landmasses (map, asset, id, fortified_by) VALUES ('world', 'borders.bin', 4, 'fr'), ('battle', 'borders.bin', 4, 'it');
	`)
	require.NoError(t, err)

	down, err := fs.ReadFile(migrations.FS, arenas+"_arenas.down.sql")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.Equal(t, []int{1}, scan(t, db, `SELECT id FROM tiles`))
	assert.Equal(t, map[int]string{4: "fr"}, locksOf(t, db, "borders.bin"))
}

func TestTheOutboxGoesWithTheArenasEventsItHeld(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, dropOutbox))
	_, err := db.On(t.Context()).ExecContext(t.Context(), tellArenas)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	assert.Equal(t, 0, countWhere(t, db, `SELECT count(*) WHERE to_regclass('outbox') IS NOT NULL`))
	assert.Equal(t, 0, countWhere(t, db, `SELECT count(*) WHERE to_regtype('outbox_event') IS NOT NULL`))
}
