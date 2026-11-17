package migrations_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

const (
	beforeBoards  = 20261116191131
	boardsAndEras = 20261117133227
)

// The seed's closed arena opened at 18:00:00.5 and closed at 20:00:01.7; the other opened at 10:00 and still plays.
var (
	siegeOpens = time.Date(2026, 11, 14, 18, 0, 0, 500_000_000, time.UTC)
	openOpens  = time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
)

// The one test of a migration's rows: it moved each arena's events to its map, the last time an arena's window was read
// by time, so a wrong window would misfile history for good.
func TestBoardsAndErasNameEarth1EverywhereAndMoveEachArenasEventsToItsMap(t *testing.T) {
	ctx := t.Context()
	db := migratedTo(t, beforeBoards)
	seedBeforeBoards(ctx, t, db)

	require.NoError(t, db.MigrateTo(ctx, migrations.FS, boardsAndEras))

	var board struct {
		id   int
		name string
		hash string
	}
	require.NoError(t, db.On(ctx).QueryRowContext(ctx, `SELECT id, name, hash FROM boards`).
		Scan(&board.id, &board.name, &board.hash))
	assert.Equal(t, 1, board.id)
	assert.Equal(t, "earth-1", board.name)
	assert.Equal(t, "f7db567f", board.hash, "the hash of map/boards/earth-1/board-<hash>.json")

	assert.Equal(t, []era{
		{Map: "world", Board: 1, StartedAt: time.Unix(0, 0).UTC()},
		{Map: "siege", Board: 1, StartedAt: siegeOpens},
		{Map: "open", Board: 1, StartedAt: openOpens},
	}, erasOf(ctx, t, db))

	for _, table := range []string{"tiles", "landmasses", "mines", "viruses", "ledger_events"} {
		var boards []int
		rows, err := db.On(ctx).QueryContext(ctx, `SELECT DISTINCT board FROM `+table)
		require.NoError(t, err)
		for rows.Next() {
			var id int
			require.NoError(t, rows.Scan(&id))
			boards = append(boards, id)
		}
		require.NoError(t, rows.Err())
		_ = rows.Close()
		assert.Equal(t, []int{1}, boards, "every row of %s names earth-1", table)
	}

	assert.Equal(t, map[int64]string{
		1: "world", // before the siege opened
		2: "world", // in the second it opened, before it
		3: "siege",
		4: "siege", // in the second before the one it closed in
		5: "world", // in the second it closed: the freeze kept the siege's own takes off it
		6: "world",
		7: "open", // the arena still open
	}, mapsOf(ctx, t, db))

	_, err := db.On(ctx).ExecContext(ctx, `INSERT INTO tiles (map, id, country, shields) VALUES ('world', 9, 'fr', 0)`)
	require.Error(t, err, "a tile that names no board is refused: the default went")
	_, err = db.On(ctx).ExecContext(ctx, `
		INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at, map, board)
		VALUES (8, 'take', 1, '', 'fr', '', now(), 'world', 2)
	`)
	require.Error(t, err, "an event on a board planet does not keep is refused")

	require.NoError(t, db.MigrateTo(ctx, migrations.FS, beforeBoards))

	var left int
	require.NoError(t, db.On(ctx).QueryRowContext(ctx, `
		SELECT count(*) FROM information_schema.columns
		WHERE table_schema = 'planet'
			AND (column_name = 'board' OR column_name = 'map' AND table_name = 'ledger_events')
	`).Scan(&left))
	assert.Zero(t, left, "the way back drops every board and the ledger's map")
	require.NoError(t, db.On(ctx).QueryRowContext(ctx, `
		SELECT count(*) FROM information_schema.tables WHERE table_schema = 'planet' AND table_name IN ('boards', 'eras')
	`).Scan(&left))
	assert.Zero(t, left)
	var events int
	require.NoError(t, db.On(ctx).QueryRowContext(ctx, `SELECT count(*) FROM ledger_events`).Scan(&events))
	assert.Equal(t, 7, events, "the way back keeps every event")

	require.NoError(t, db.MigrateTo(ctx, migrations.FS, boardsAndEras), "and it runs again after its way back")
	assert.Equal(t, "siege", mapsOf(ctx, t, db)[3])
}

func migratedTo(t *testing.T, version uint) *cppg.Postgres {
	t.Helper()
	server := cppgtest.StartServer(t)
	db := cppg.New(server.ConfigFor("planet"))
	require.NoError(t, db.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.MigrateTo(t.Context(), migrations.FS, version))
	return db
}

func seedBeforeBoards(ctx context.Context, t *testing.T, db *cppg.Postgres) {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO tiles (map, id, country, shields) VALUES ('world', 1, 'fr', 2), ('open', 2, 'de', 50)`,
		`INSERT INTO landmasses (map, asset, id, fortified_by) VALUES ('world', 'borders-e6851c0e.bin', 3, 'fr')`,
		`INSERT INTO mines (map, tile, flag, account, scope) VALUES ('world', 1, 'fr', gen_random_uuid(), '1.2.3.4')`,
		`INSERT INTO viruses (map, id, account, scope, flag, victim, tile, infected, infections, next_at)
			VALUES ('world', gen_random_uuid(), gen_random_uuid(), '1.2.3.4', 'fr', 'de', 1, '{1}', 1, now())`,
		`INSERT INTO arenas (id, ends_at, opened_at, freeze_ms) VALUES
			('siege', '2026-11-14 20:00:00+00', '2026-11-14 18:00:00.5+00', 8000),
			('open', '2026-11-17 12:00:00+00', '2026-11-17 10:00:00+00', 8000)`,
		`INSERT INTO closed_arenas (id, closed_at) VALUES ('siege', '2026-11-14 20:00:01.7+00')`,
		`INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at) VALUES
			(1, 'take', 1, '', 'fr', '', '2026-11-14 17:59:59+00'),
			(2, 'take', 1, '', 'fr', '', '2026-11-14 18:00:00+00'),
			(3, 'take', 1, '', 'de', 'fr', '2026-11-14 18:00:09+00'),
			(4, 'take', 1, '', 'fr', 'de', '2026-11-14 20:00:00.9+00'),
			(5, 'take', 1, '', 'de', 'fr', '2026-11-14 20:00:01+00'),
			(6, 'take', 1, '', 'fr', 'de', '2026-11-16 09:00:00+00'),
			(7, 'take', 2, '', 'fr', 'de', '2026-11-17 10:00:09+00')`,
	} {
		_, err := db.On(ctx).ExecContext(ctx, statement)
		require.NoError(t, err, statement)
	}
}

type era struct {
	Map       string
	Board     int
	StartedAt time.Time
}

func erasOf(ctx context.Context, t *testing.T, db *cppg.Postgres) []era {
	t.Helper()
	rows, err := db.On(ctx).QueryContext(ctx, `SELECT map, board, started_at FROM eras ORDER BY started_at`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var eras []era
	for rows.Next() {
		var read era
		require.NoError(t, rows.Scan(&read.Map, &read.Board, &read.StartedAt))
		read.StartedAt = read.StartedAt.UTC()
		eras = append(eras, read)
	}
	require.NoError(t, rows.Err())
	return eras
}

func mapsOf(ctx context.Context, t *testing.T, db *cppg.Postgres) map[int64]string {
	t.Helper()
	rows, err := db.On(ctx).QueryContext(ctx, `SELECT position, map FROM ledger_events`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	maps := map[int64]string{}
	for rows.Next() {
		var (
			position int64
			played   string
		)
		require.NoError(t, rows.Scan(&position, &played))
		maps[position] = played
	}
	require.NoError(t, rows.Err())
	return maps
}
