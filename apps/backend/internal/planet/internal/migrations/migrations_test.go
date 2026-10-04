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

	down, err := fs.ReadFile(migrations.FS, remapTiles+"_remap_tiles.down.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	ids := scan(t, db, `SELECT id FROM tiles ORDER BY id`)
	assert.ElementsMatch(t, keys(moved), ids, "the tiles that survived come back to their old ids")
}

const keepEveryTake = "20261004120000"

func TestKeepingEveryTakeGoesBackWithoutTheTakesThatLostTheirScope(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	at := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO ledger_takes (position, tile, scope, country, previous, taken_at)
		 VALUES (0, 1, NULL, 'fr', '', $1), (1, 2, '203.0.113.7', 'fr', '', $1)`, at)
	require.NoError(t, err)

	down, err := fs.ReadFile(migrations.FS, keepEveryTake+"_keep_every_take.down.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.Equal(t, []int{2}, scan(t, db, `SELECT tile FROM ledger_takes`))
	_, err = db.ExecContext(t.Context(),
		`INSERT INTO ledger_takes (position, tile, scope, country, previous, taken_at) VALUES (2, 3, NULL, 'fr', '', $1)`, at)
	assert.Error(t, err, "a scope is required again")
}

const takeFeed = "20261005120000"

const ada = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

func take(t *testing.T, db *cppg.Postgres, position int, scope, account string) {
	t.Helper()

	_, err := db.ExecContext(t.Context(),
		`INSERT INTO ledger_takes (position, tile, scope, account, country, previous, taken_at)
		 VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, '')::uuid, 'fr', '', $5)`,
		position, position+1, scope, account, time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
}

func feedStart(t *testing.T, db *cppg.Postgres) int {
	t.Helper()

	starts := scan(t, db, `SELECT start FROM ledger_feed`)
	require.Len(t, starts, 1)
	return starts[0]
}

func TestTheFeedStartsWhereTheFirstBootWithItGivesItsFirstTake(t *testing.T) {
	for name, c := range map[string]struct {
		takes []int
		head  int
		start int
	}{
		"after the last take kept":                   {takes: []int{0, 1, 2, 3, 4}, head: 2, start: 5},
		"at the head when the window is empty":       {takes: []int{0, 1, 2}, head: 7, start: 7},
		"at zero on a ledger that never took a tile": {takes: nil, head: -1, start: 0},
	} {
		t.Run(name, func(t *testing.T) {
			db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, takeFeed))
			for _, position := range c.takes {
				take(t, db, position, "203.0.113.7", "")
			}
			if c.head >= 0 {
				_, err := db.ExecContext(t.Context(), `INSERT INTO ledger_head (head) VALUES ($1)`, c.head)
				require.NoError(t, err)
			}

			require.NoError(t, db.Migrate(t.Context(), migrations.FS))

			assert.Equal(t, c.start, feedStart(t, db))
		})
	}
}

func TestTheTakesTheMarksStillCoverAreRevertedByTheMigration(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, takeFeed))
	take(t, db, 0, "", ada)
	take(t, db, 1, "bot", "")
	take(t, db, 2, "player", ada)
	take(t, db, 3, "bot", "")
	take(t, db, 4, "player", "")
	_, err := db.ExecContext(t.Context(), `INSERT INTO ledger_head (head) VALUES (1)`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `INSERT INTO ledger_forgotten (scope, before_position) VALUES ('bot', 2)`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `INSERT INTO ledger_forgotten_accounts (account, before_position) VALUES ($1, 5)`, ada)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	assert.Equal(t, []int{1, 2}, scan(t, db, `SELECT position FROM ledger_takes WHERE reverted ORDER BY position`),
		"the scope's take before its mark, and the account's take inside the window")
}

func TestTheFeedGoesBack(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	down, err := fs.ReadFile(migrations.FS, takeFeed+"_take_feed.down.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.Empty(t, scan(t, db, `SELECT count(*) FROM information_schema.columns WHERE table_name = 'ledger_takes' AND column_name = 'reverted' HAVING count(*) > 0`))
	assert.Empty(t, scan(t, db, `SELECT count(*) FROM information_schema.tables WHERE table_name = 'ledger_feed' HAVING count(*) > 0`))
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
