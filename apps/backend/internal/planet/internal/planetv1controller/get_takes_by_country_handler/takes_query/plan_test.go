package takes_query

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheTakesOfOneAccountAreReadFromTheIndexAlone(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at, account)
		SELECT n, 'take'::ledger_event_kind, n % 262119 + 1, NULL, 'fr', 'de', now(), md5((n % 500)::text)::uuid
		FROM generate_series(0, 199999) n`)
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), `VACUUM ANALYZE ledger_events`)
	require.NoError(t, err)
	var account uuid.UUID
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `SELECT md5('7')::uuid`).Scan(&account))

	plan := explain(t, db, account)

	assert.Contains(t, plan, "Index Only Scan using ledger_events_account_kind", plan)
	assert.NotContains(t, plan, "Seq Scan", plan)
}

func explain(t *testing.T, db *cppg.Postgres, account uuid.UUID) string {
	t.Helper()

	rows, err := db.On(t.Context()).QueryContext(t.Context(), "EXPLAIN "+takes, account)
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
