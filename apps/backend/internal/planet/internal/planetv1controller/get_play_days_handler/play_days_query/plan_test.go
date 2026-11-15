package play_days_query_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_play_days_handler/play_days_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

func TestAPageOfAccountsIsFoundThroughTheAccountIndex(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at, account)
		SELECT n, 'take'::ledger_event_kind, n % 260503 + 1, NULL, 'fr', 'de', now() - (n % 40) * interval '1 day',
			md5((n % 5000)::text)::uuid
		FROM generate_series(0, 199999) n`)
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), `VACUUM ANALYZE ledger_events`)
	require.NoError(t, err)
	var accounts []string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT array_agg(md5(n::text)::uuid::text) FROM generate_series(0, 499) n`).Scan(pq.Array(&accounts)))

	plan := explain(t, db, accounts)

	assert.Contains(t, plan, "ledger_events_account_kind", plan)
	assert.NotContains(t, plan, "Seq Scan", plan)
}

func explain(t *testing.T, db *cppg.Postgres, accounts []string) string {
	t.Helper()

	for _, account := range accounts {
		_, err := uuid.Parse(account)
		require.NoError(t, err)
	}
	rows, err := db.On(t.Context()).QueryContext(t.Context(), "EXPLAIN "+play_days_query.PlayDays, pq.Array(accounts))
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
