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

const (
	ledgerKnockOut       = "20261111140100"
	knockOutsOnEveryMap  = "20261111150100"
	insertKnockOutOfFrom = `INSERT INTO knockouts (map, flag, by_flag, knocked_out_at) VALUES ($1, 'fr', 'de', now())`
)

func TestAFlagOutIsKeptOncePerMapTheWorldsToo(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	_, err := db.On(t.Context()).ExecContext(t.Context(), insertKnockOutOfFrom, "battle")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), insertKnockOutOfFrom, "battle")
	require.Error(t, err, "a flag is out of one map once")
	_, err = db.On(t.Context()).ExecContext(t.Context(), insertKnockOutOfFrom, "world")
	require.NoError(t, err, "the knock-out is a rule of its own, on any map")
}

func TestTheWorldsFlagsOutGoWithTheWayBack(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	for _, played := range []string{"world", "battle"} {
		_, err := db.On(t.Context()).ExecContext(t.Context(), insertKnockOutOfFrom, played)
		require.NoError(t, err)
	}

	down, err := fs.ReadFile(migrations.FS, knockOutsOnEveryMap+"_knockouts_on_every_map.down.sql")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.Equal(t, 1, countWhere(t, db, `SELECT count(*) FROM knockouts WHERE map = 'battle'`))
	assert.Equal(t, 0, countWhere(t, db, `SELECT count(*) FROM knockouts WHERE map = 'world'`))
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
