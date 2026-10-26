package migrations_test

import (
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const remapTiles = "20260921120000"

func before(t *testing.T, version string) fs.FS {
	t.Helper()

	return cppg.MigrationsBefore(t, migrations.FS, version)
}

var moved = map[int]int{1: 1, 90: 89, 415: 415, 416: 418, 417: 420, 257947: 262119}

var gone = []int{89, 257948}

func seed(t *testing.T, db *cppg.Postgres) {
	t.Helper()

	at := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	position := 0
	for _, old := range append(keys(moved), gone...) {
		_, err := db.On(t.Context()).ExecContext(t.Context(),
			`INSERT INTO tiles (id, country) VALUES ($1, $2)`, old, "fr")
		require.NoError(t, err)

		_, err = db.On(t.Context()).ExecContext(t.Context(),
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
		require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
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

	tiles := scan(t, db, `SELECT tile FROM ledger_events ORDER BY tile`)
	assert.ElementsMatch(t, values(moved), tiles,
		"a take on a tile the new map does not have is deleted, not left pointing at other ground")
}

func TestTheRemapGoesBack(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "planet", before(t, remapTiles))
	seed(t, db)

	require.NoError(t, db.Migrate(t.Context(), before(t, ledgerEvents)))

	down, err := fs.ReadFile(migrations.FS, remapTiles+"_remap_tiles.down.sql")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	ids := scan(t, db, `SELECT id FROM tiles ORDER BY id`)
	assert.ElementsMatch(t, keys(moved), ids, "the tiles that survived come back to their old ids")
}

const keepEveryTake = "20261004120000"

func TestKeepingEveryTakeGoesBackWithoutTheTakesThatLostTheirScope(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, ledgerEvents))
	at := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO ledger_takes (position, tile, scope, country, previous, taken_at)
		 VALUES (0, 1, NULL, 'fr', '', $1), (1, 2, '203.0.113.7', 'fr', '', $1)`, at)
	require.NoError(t, err)

	down, err := fs.ReadFile(migrations.FS, keepEveryTake+"_keep_every_take.down.sql")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.Equal(t, []int{2}, scan(t, db, `SELECT tile FROM ledger_takes`))
	_, err = db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO ledger_takes (position, tile, scope, country, previous, taken_at) VALUES (2, 3, NULL, 'fr', '', $1)`, at)
	assert.Error(t, err, "a scope is required again")
}

const ledgerEvents = "20261006130000"

func TestEveryTakeKeptBeforeTheEventsIsATake(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, ledgerEvents))
	at := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO ledger_takes (position, tile, scope, country, previous, taken_at)
		 VALUES (0, 1, NULL, 'fr', '', $1), (1, 2, '203.0.113.7', 'fr', 'de', $1)`, at)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), before(t, ledgerEventKind)))

	assert.Equal(t, []int{2}, scan(t, db,
		`SELECT count(*) FROM ledger_events WHERE kind = 'take' AND payload IS NULL`))

	insert := `INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at, payload)
		VALUES ($1, $2, 3, '203.0.113.7', '', '', $3, $4)`
	_, err = db.On(t.Context()).ExecContext(t.Context(), insert, 2, "", at, nil)
	require.Error(t, err, "an event has a kind")
	_, err = db.On(t.Context()).ExecContext(t.Context(), insert, 3, "bomb", at, `{"cleared": []}`)
	require.NoError(t, err, "which the table leaves to the ledger")
}

func TestTheEventsGoBackToTakesWithoutTheBombs(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, ledgerEventKind))
	at := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at, payload)
		 VALUES (0, 'take', 1, '203.0.113.7', 'fr', '', $1, NULL), (1, 'bomb', 1, '203.0.113.7', '', 'fr', $1, '{}')`, at)
	require.NoError(t, err)

	down, err := fs.ReadFile(migrations.FS, ledgerEvents+"_ledger_events.down.sql")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.Equal(t, []int{0}, scan(t, db, `SELECT position FROM ledger_takes`))
}

const ledgerEventKind = "20261012120000"

func TestEveryEventKeptBeforeTheTypeKeepsItsKind(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, ledgerEventKind))
	at := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	kinds := []string{"take", "strike", "spread", "enclose", "bomb", "shield", "fortify"}
	insert := `INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at)
		VALUES ($1, $2, 1, NULL, 'fr', '', $3)`
	for position, kind := range kinds {
		_, err := db.On(t.Context()).ExecContext(t.Context(), insert, position, kind, at)
		require.NoError(t, err)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	for position, kind := range kinds {
		var kept string
		require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
			`SELECT kind FROM ledger_events WHERE position = $1`, position).Scan(&kept))
		assert.Equal(t, kind, kept)
	}
	_, err := db.On(t.Context()).ExecContext(t.Context(), insert, len(kinds), "quake", at)
	require.Error(t, err, "a kind is a value of the type before an event has it")
	_, err = db.On(t.Context()).ExecContext(t.Context(), insert, len(kinds), "", at)
	require.Error(t, err, "an event still has a kind")
}

const ledgerEventsAccountKind = "20261017120000"

func TestTheAccountKindIndexRunsAgainWhenItsVersionIsForcedBack(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	up, err := fs.ReadFile(migrations.FS, ledgerEventsAccountKind+"_ledger_events_account_kind.up.sql")
	require.NoError(t, err)

	_, err = db.On(t.Context()).ExecContext(t.Context(), string(up))

	require.NoError(t, err, "a run killed after the build committed leaves the index there")
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

	rows, err := db.On(t.Context()).QueryContext(t.Context(), query)
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

const ledgerRelayed = "20261025180000"

func TestTheLedgerIsToldFromTheFirstEventTheOldServerDidNotPublish(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, ledgerRelayed))
	for position := range 3 {
		_, err := db.On(t.Context()).ExecContext(t.Context(), `
			INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at)
			VALUES ($1, 'take', 1, '203.0.113.7', 'fr', '', now())
		`, position)
		require.NoError(t, err)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	var relayed int64
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `SELECT position FROM ledger_relayed`).Scan(&relayed))
	assert.Equal(t, int64(3), relayed, "it published every take it made, at the take")
}

const tileHolders = "20261026120000"

func TestEachTileIsHeldByTheAccountOfTheLastChangeThatGaveItItsFlag(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, tileHolders))
	ana, bob := "0192f0a0-0000-7000-8000-000000000001", "0192f0a0-0000-7000-8000-000000000002"
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO tiles (id, country) VALUES (1, 'fr'), (2, 'de'), (3, 'fr'), (4, 'it'), (5, 'fr')
	`)
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO ledger_events (position, kind, tile, scope, account, country, previous, taken_at, payload) VALUES
			(0, 'take', 1, '203.0.113.7', $1, 'fr', '', now(), NULL),
			(1, 'take', 1, '203.0.113.7', $2, 'fr', 'fr', now(), NULL),
			(2, 'take', 2, '203.0.113.7', $1, 'fr', '', now(), NULL),
			(3, 'spread', 9, '203.0.113.7', $2, 'fr', '', now(), '{"taken": [{"tile": 3}]}'),
			(4, 'take', 4, '203.0.113.7', $1, 'it', '', now(), NULL),
			(5, 'bomb', 4, '203.0.113.7', $2, 'fr', '', now(), '{"flag": "fr", "point": [0, 0, 1], "radius": 0.01, "cleared": [{"tile": 4, "owner": "it"}]}'),
			(6, 'take', 5, '203.0.113.7', NULL, 'fr', '', now(), NULL)
	`, ana, bob)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	rows, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT id, COALESCE(holder::text, '') FROM tiles ORDER BY id`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	holders := map[int]string{}
	for rows.Next() {
		var (
			id     int
			holder string
		)
		require.NoError(t, rows.Scan(&id, &holder))
		holders[id] = holder
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, map[int]string{1: bob, 2: "", 3: bob, 4: "", 5: ""}, holders,
		"the last take; none once another flag was written; a spread's tile; none after a clear; none for no account")
}
