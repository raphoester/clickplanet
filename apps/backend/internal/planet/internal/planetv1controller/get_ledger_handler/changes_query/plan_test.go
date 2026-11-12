package changes_query_test

import (
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler/changes_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

func TestAPageIsFoundFromTheHeadByThePrimaryKeyAndSortedOnlyWithinEachEvent(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)
	for _, statement := range []string{
		`INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at)
		SELECT n, 'take'::ledger_event_kind, n % 260503 + 1, '203.0.113.7', 'fr', 'de', now() - (200000 - n) * interval '1 second'
		FROM generate_series(0, 199999) n`,
		`INSERT INTO ledger_head (singleton, head) VALUES (true, 190000) ON CONFLICT (singleton) DO UPDATE SET head = 190000`,
		`VACUUM ANALYZE ledger_events`,
	} {
		_, err := db.On(t.Context()).ExecContext(t.Context(), statement)
		require.NoError(t, err)
	}

	arena := []time.Time{time.Now().Add(-30 * time.Minute)}
	rows, err := db.On(t.Context()).QueryContext(t.Context(), "EXPLAIN "+changes_query.Changes, time.Now().Add(-time.Hour), time.Now(), 0, changes_query.PageEvents,
		pq.Array(arena), pq.Array([]time.Time{time.Now().Add(-10 * time.Minute)}))
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var lines []string
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		lines = append(lines, line)
	}
	require.NoError(t, rows.Err())
	plan := strings.Join(lines, "\n")

	assert.Contains(t, plan, "ledger_events_pkey", plan)
	assert.NotContains(t, plan, "Seq Scan on ledger_events", plan)
	assert.Contains(t, plan, "Incremental Sort", plan)
}
