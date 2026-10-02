package migrations_test

import (
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const remapTiles = "20260921120000"

// before is the migrations older than version, so a test can write rows the way they were.
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

// The ids are read off the runs the generated migration carries: 1 and 415 do not move, 90 and 416
// shift by what was added or removed before them, and 89 and 257948 were on open sea in the new map.
var moved = map[int]int{1: 1, 90: 89, 415: 415, 416: 418, 417: 420, 257947: 262119}

var gone = []int{89, 257948}

func seed(t *testing.T, db *cppg.Postgres) {
	t.Helper()

	at := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	position := 0
	for _, old := range append(keys(moved), gone...) {
		_, err := db.ExecContext(t.Context(),
			`INSERT INTO tiles (id, country) VALUES ($1, $2)`, old, "fr")
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(),
			`INSERT INTO ledger_takes (position, tile, scope, country, previous, taken_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			position, old, "203.0.113.7", "fr", "de", at)
		require.NoError(t, err)
		position++
	}
}

func TestTheRemapFollowsEveryOwnedTileToItsNewID(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "planet", before(t, remapTiles))
	seed(t, db)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	ids := scan(t, db, `SELECT id FROM tiles ORDER BY id`)
	assert.ElementsMatch(t, values(moved), ids, "every surviving tile is at its new id")

	for _, id := range ids {
		var country string
		require.NoError(t, db.QueryRowContext(t.Context(),
			`SELECT country FROM tiles WHERE id = $1`, id).Scan(&country))
		assert.Equal(t, "fr", country, "tile %d keeps its owner", id)
	}
}

func TestTheRemapDropsTheTilesTheNewMapDoesNotHave(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "planet", before(t, remapTiles))
	seed(t, db)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	ids := scan(t, db, `SELECT id FROM tiles ORDER BY id`)
	assert.Len(t, ids, len(moved), "the two tiles on open sea are gone, and nothing else")
}

func TestTheRemapMovesTheLedgerAndForgetsTakesOnTilesThatWent(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "planet", before(t, remapTiles))
	seed(t, db)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	tiles := scan(t, db, `SELECT tile FROM ledger_takes ORDER BY tile`)
	assert.ElementsMatch(t, values(moved), tiles,
		"a take on a tile the new map does not have is deleted, not left pointing at other ground")
}

func TestTheRemapGoesBack(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "planet", before(t, remapTiles))
	seed(t, db)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	// Run the down file the way golang-migrate would, rather than teaching cppg to migrate down for
	// one test: lib/pq sends it over the simple protocol, so the whole file is one transaction.
	down, err := fs.ReadFile(migrations.FS, remapTiles+"_remap_tiles.down.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	ids := scan(t, db, `SELECT id FROM tiles ORDER BY id`)
	assert.ElementsMatch(t, keys(moved), ids, "the tiles that survived come back to their old ids")
}

func keys(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func values(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func scan(t *testing.T, db *cppg.Postgres, query string) []int {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), query)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()

	var out []int
	for rows.Next() {
		var value int
		require.NoError(t, rows.Scan(&value))
		out = append(out, value)
	}
	require.NoError(t, rows.Err())
	return out
}
