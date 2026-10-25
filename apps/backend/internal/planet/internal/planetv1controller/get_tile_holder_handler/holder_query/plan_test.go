package holder_query

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheHolderOfOneTileIsFoundThroughTheIndexes(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at, account, payload)
		SELECT n,
			CASE n % 10 WHEN 0 THEN 'spread' WHEN 1 THEN 'bomb' ELSE 'take' END::ledger_event_kind,
			n % 262119 + 1, NULL, CASE n % 10 WHEN 1 THEN '' ELSE 'fr' END, 'de', now(), md5((n % 500)::text)::uuid,
			CASE n % 10
				WHEN 0 THEN jsonb_build_object('taken', jsonb_build_array(jsonb_build_object('tile', (n + 7) % 262119 + 1, 'owner', 'de')))
				WHEN 1 THEN jsonb_build_object('cleared', jsonb_build_array(jsonb_build_object('tile', (n + 3) % 262119 + 1, 'owner', 'de')))
			END
		FROM generate_series(0, 199999) n`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `VACUUM ANALYZE ledger_events`)
	require.NoError(t, err)

	plan := explain(t, db, 4242)

	assert.Contains(t, plan, "ledger_events_tile_take", plan)
	assert.Contains(t, plan, "ledger_events_tile_payload", plan)
	assert.NotContains(t, plan, "Seq Scan", plan)
}

func explain(t *testing.T, db *cppg.Postgres, tile uint32) string {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), "EXPLAIN "+lastChange, tile)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()

	var lines []string
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		lines = append(lines, line)
	}
	require.NoError(t, rows.Err())
	return strings.Join(lines, "\n")
}
