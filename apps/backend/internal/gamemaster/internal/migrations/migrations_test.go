package migrations_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const halloweenDropped = "20261110150000"

func TestTheHalloweenEntryPlaysOnAsAPlainOne(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", cppg.MigrationsBefore(t, migrations.FS, halloweenDropped))
	at := time.Date(2026, 10, 31, 14, 35, 0, 0, time.UTC)
	insert := `INSERT INTO schedule_entries (id, rule_set, starts_at, ends_at, priority, made_at, kind, times)
		VALUES ($1, 'default', $2, $3, 1, $2, $4, 0)`
	_, err := db.On(t.Context()).ExecContext(t.Context(), insert, uuid.UUID{15: 1}, at, at.Add(time.Hour), "halloween")
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	var kind string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT kind FROM schedule_entries WHERE id = $1`, uuid.UUID{15: 1}).Scan(&kind))
	assert.Equal(t, "plain", kind)
	_, err = db.On(t.Context()).ExecContext(t.Context(), insert, uuid.UUID{15: 2}, at, at.Add(time.Hour), "halloween")
	require.Error(t, err, "no entry is of kind halloween any more")
}
