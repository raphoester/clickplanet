package landmass_query

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheLastFortifyIsFoundThroughItsIndex(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at, account, payload)
		SELECT n,
			CASE WHEN n % 1000 = 0 THEN 'fortify' ELSE 'take' END::ledger_event_kind,
			n % 262119 + 1, NULL, 'fr', 'de', now(), md5((n % 500)::text)::uuid,
			CASE WHEN n % 1000 = 0 THEN jsonb_build_object('landmass', n % 657 + 1, 'raised', '[]'::jsonb) END
		FROM generate_series(0, 199999) n`)
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), `VACUUM ANALYZE ledger_events`)
	require.NoError(t, err)

	rows, err := db.On(t.Context()).QueryContext(t.Context(), "EXPLAIN "+lastFortify, 42)
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

	assert.Contains(t, plan, "ledger_events_fortify", plan)
	assert.NotContains(t, plan, "Seq Scan", plan)
}
