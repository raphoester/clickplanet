package changes_query

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheWindowIsFoundFromTheHeadByThePrimaryKey(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	for _, statement := range []string{
		`INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at)
		SELECT n, 'take'::ledger_event_kind, n % 262119 + 1, '203.0.113.7', 'fr', 'de', now() - (200000 - n) * interval '1 second'
		FROM generate_series(0, 199999) n`,
		`INSERT INTO ledger_head (singleton, head) VALUES (true, 190000) ON CONFLICT (singleton) DO UPDATE SET head = 190000`,
		`VACUUM ANALYZE ledger_events`,
	} {
		_, err := db.On(t.Context()).ExecContext(t.Context(), statement)
		require.NoError(t, err)
	}

	rows, err := db.On(t.Context()).QueryContext(t.Context(), "EXPLAIN "+changes, time.Now().Add(-time.Hour), time.Now())
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
}
