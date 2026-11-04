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

	require.NoError(t, db.Migrate(t.Context(), before(t, seatsAndLakes)))

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

	require.NoError(t, db.Migrate(t.Context(), before(t, seatsAndLakes)))

	ids := scan(t, db, `SELECT id FROM tiles ORDER BY id`)
	assert.Len(t, ids, len(moved), "the two tiles on open sea are gone, and nothing else")
}

func TestTheRemapMovesTheLedgerAndForgetsTakesOnTilesThatWent(t *testing.T) {
	server := cppg.StartTestServer(t)
	db := server.OpenSchema(t, "planet", before(t, remapTiles))
	seed(t, db)

	require.NoError(t, db.Migrate(t.Context(), before(t, seatsAndLakes)))

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

const tileHolders = "20261026180000"

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

const seatsAndLakes = "20261103120000"

// Old wire id to new across the remap that seats every country and opens the lakes: the first tile, one past the
// first seat, one past the first lakes, the last.
var seatMoved = map[int]int{1: 1, 23428: 23334, 100336: 99217, 262119: 260503}

// A tile in the middle of Lake Superior, which the new map does not have.
const onLakeSuperior = 90992

const (
	bordersBeforeSeats = "borders-67e352b1.bin"
	bordersWithSeats   = "borders-e6851c0e.bin"
)

func TestTheSeatsAndLakesRemapCarriesEveryColumnOfATile(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, seatsAndLakes))
	ana := "0192f0a0-0000-7000-8000-000000000001"
	for old := range seatMoved {
		_, err := db.On(t.Context()).ExecContext(t.Context(),
			`INSERT INTO tiles (id, country, shields, holder) VALUES ($1, 'fr', 3, $2)`, old, ana)
		require.NoError(t, err)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	rows, err := db.On(t.Context()).QueryContext(t.Context(),
		`SELECT id, country, shields, COALESCE(holder::text, '') FROM tiles`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var ids []int
	for rows.Next() {
		var (
			id, shields     int
			country, holder string
		)
		require.NoError(t, rows.Scan(&id, &country, &shields, &holder))
		ids = append(ids, id)
		assert.Equal(t, []any{"fr", 3, ana}, []any{country, shields, holder}, "tile %d keeps its flag, shields and holder", id)
	}
	require.NoError(t, rows.Err())
	assert.ElementsMatch(t, values(seatMoved), ids)
}

func TestTheSeatsAndLakesRemapMovesEveryTileTheLedgerNamesAndTheLandmassAFortifyNames(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, seatsAndLakes))
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at, payload) VALUES
			(0, 'take', 262119, '203.0.113.7', 'fr', '', now(), NULL),
			(1, 'spread', 23428, '203.0.113.7', 'fr', '', now(), '{"taken": [{"tile": 100336}]}'),
			(2, 'fortify', 262119, '203.0.113.7', 'fr', '', now(),
				'{"landmass": 658, "ground": "fr", "tiles": 3, "raised": [{"tile": 262119, "shields": 1}]}')
	`)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	assert.Equal(t, []int{260503, 23334, 260503}, scan(t, db, `SELECT tile FROM ledger_events ORDER BY position`))
	assert.Equal(t, []int{99217}, scan(t, db,
		`SELECT (payload->'taken'->0->>'tile')::integer FROM ledger_events WHERE position = 1`))
	assert.Equal(t, []int{260503}, scan(t, db,
		`SELECT (payload->'raised'->0->>'tile')::integer FROM ledger_events WHERE position = 2`))
	assert.Equal(t, []int{694}, scan(t, db,
		`SELECT (payload->>'landmass')::integer FROM ledger_events WHERE position = 2`),
		"the territory card reads the last fortify of a landmass by its id in the new blob")
}

func seedLocks(t *testing.T, db *cppg.Postgres) {
	t.Helper()

	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO landmasses (asset, id, fortified_by) VALUES ($1, 1, 'fr'), ($1, 658, 'de')`, bordersBeforeSeats)
	require.NoError(t, err)
}

func locksOf(t *testing.T, db *cppg.Postgres, asset string) map[int]string {
	t.Helper()

	rows, err := db.On(t.Context()).QueryContext(t.Context(),
		`SELECT id, fortified_by FROM landmasses WHERE asset = $1`, asset)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	locks := map[int]string{}
	for rows.Next() {
		var (
			id   int
			flag string
		)
		require.NoError(t, rows.Scan(&id, &flag))
		locks[id] = flag
	}
	require.NoError(t, rows.Err())
	return locks
}

func TestTheSeatsAndLakesRemapKeepsEveryFortifyLockOnTheNewBlob(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, seatsAndLakes))
	seedLocks(t, db)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	assert.Equal(t, map[int]string{1: "fr", 694: "de"}, locksOf(t, db, bordersWithSeats),
		"the boot reads the locks of the blob it loads, so they are copied to its name")
}

func TestTheSeatsAndLakesRemapGoesBack(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, seatsAndLakes))
	for old := range seatMoved {
		_, err := db.On(t.Context()).ExecContext(t.Context(), `INSERT INTO tiles (id, country) VALUES ($1, 'fr')`, old)
		require.NoError(t, err)
	}
	seedLocks(t, db)
	require.NoError(t, db.Migrate(t.Context(), migrations.FS))
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE landmasses SET fortified_by = 'it' WHERE asset = $1 AND id = 694`, bordersWithSeats)
	require.NoError(t, err)

	down, err := fs.ReadFile(migrations.FS, seatsAndLakes+"_remap_tiles_seats_and_lakes.down.sql")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.ElementsMatch(t, keys(seatMoved), scan(t, db, `SELECT id FROM tiles`), "every tile is back at its old id")
	assert.Equal(t, map[int]string{1: "fr", 658: "it"}, locksOf(t, db, bordersBeforeSeats),
		"the locks taken since go back with the tiles")
}

func TestTheSeatsAndLakesRemapDropsATileUnderALakeAndForgetsItsTakes(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, seatsAndLakes))
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO tiles (id, country) VALUES ($1, 'us'), (1, 'fr')`, onLakeSuperior)
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at, payload) VALUES
			(0, 'take', $1, '203.0.113.7', 'us', '', now(), NULL),
			(1, 'take', 1, '203.0.113.7', 'fr', '', now(), NULL)
	`, onLakeSuperior)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	assert.Equal(t, []int{1}, scan(t, db, `SELECT id FROM tiles`), "Lake Superior is water")
	assert.Equal(t, []int{1}, scan(t, db, `SELECT tile FROM ledger_events`),
		"a take on it is deleted, not left pointing at other ground")
}
