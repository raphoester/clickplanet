package postgres_ledger_store

import (
	"strings"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestAMarkForgetsItsCallersEventsWithoutReadingTheWholeLedger(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at, account)
		SELECT n, 'take', n % 260503 + 1, CASE WHEN n < 150000 THEN NULL ELSE '203.0.113.' || (n % 250) END,
			'fr', 'de', now(), md5((n % 500)::text)::uuid
		FROM generate_series(0, 199999) n`)
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), `VACUUM ANALYZE ledger_events`)
	require.NoError(t, err)

	scopes := explain(t, db, forgetScopesEvents, pq.Array([]string{"203.0.113.7"}), pq.Array([]int64{200000}), 150000)
	accounts := explain(t, db, forgetAccountsEvents,
		pq.Array([]string{"0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"}), pq.Array([]int64{200000}))

	assert.Contains(t, scopes, "ledger_events_pkey", scopes)
	assert.NotContains(t, scopes, "Seq Scan on ledger_events", scopes)
	assert.Contains(t, accounts, "ledger_events_account_kind", accounts)
	assert.NotContains(t, accounts, "Seq Scan on ledger_events", accounts)
}

func explain(t *testing.T, db *cppg.Postgres, statement string, args ...any) string {
	t.Helper()

	rows, err := db.On(t.Context()).QueryContext(t.Context(), "EXPLAIN "+statement, args...)
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
