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

const ledgerKnockOut = "20261111140100"

func TestAFlagOutIsKeptOnAnArenasMapOnly(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	insert := `INSERT INTO knockouts (map, flag, by_flag, knocked_out_at) VALUES ($1, 'fr', 'de', now())`

	_, err := db.On(t.Context()).ExecContext(t.Context(), insert, "battle")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), insert, "battle")
	require.Error(t, err, "a flag is out of one arena once")
	_, err = db.On(t.Context()).ExecContext(t.Context(), insert, "world")
	require.Error(t, err, "the world's map knocks nobody out")
}

func TestTheKnockOutsGoBackWithTheirLedgerEvents(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	at := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO ledger_events (position, kind, tile, scope, country, previous, taken_at)
		VALUES (0, 'take', 1, NULL, 'de', 'fr', $1), (1, 'knock_out', 1, NULL, 'de', 'fr', $1)
	`, at)
	require.NoError(t, err)

	down, err := fs.ReadFile(migrations.FS, ledgerKnockOut+"_ledger_knock_out.down.sql")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.Equal(t, []int{0}, scan(t, db, `SELECT position FROM ledger_events`))
}
