package migrations_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const announcementKind = "20261012120000"

func TestEveryAnnouncementKeptBeforeTheTypeKeepsItsKind(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "chat", cppg.MigrationsBefore(t, migrations.FS, announcementKind))
	at := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	kinds := []string{"bomb", "mute", "round", "fortify"}
	insert := `INSERT INTO announcements (id, kind, payload, announced_at) VALUES ($1, $2, '{}', $3)`
	for i, kind := range kinds {
		_, err := db.On(t.Context()).ExecContext(t.Context(), insert, uuid.UUID{15: byte(i)}, kind, at)
		require.NoError(t, err)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	for i, kind := range kinds {
		var kept string
		require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
			`SELECT kind FROM announcements WHERE id = $1`, uuid.UUID{15: byte(i)}).Scan(&kept))
		assert.Equal(t, kind, kept)
	}
	_, err := db.On(t.Context()).ExecContext(t.Context(), insert, uuid.UUID{15: byte(len(kinds))}, "meteor", at)
	require.Error(t, err, "a kind is a value of the type before an announcement has it")
}
