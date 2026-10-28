package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const briefsPerAccount = "20261028120000"

func TestEachSheetKeptBeforeBriefsPerAccountKeepsItsFlagsBrief(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "missions", cppg.MigrationsBefore(t, migrations.FS, briefsPerAccount))
	ada, mission := uuid.UUID{15: 1}, uuid.UUID{15: 9}
	seed := func(query string, args ...any) {
		_, err := db.On(t.Context()).ExecContext(t.Context(), query, args...)
		require.NoError(t, err)
	}
	seed(`INSERT INTO mission_briefs (day, flag, missions) VALUES ('2026-10-28', 'fr', '[{"id": "fr"}]'), ('2026-10-28', 'de', '[]')`)
	seed(`INSERT INTO mission_sheets (day, account_id, flag, counted_takes_below) VALUES ('2026-10-28', $1, 'fr', 7)`, ada)
	seed(`INSERT INTO mission_progress (day, account_id, mission_id, progress) VALUES ('2026-10-28', $1, $2, 3)`, ada, mission)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	var kept string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT missions::text FROM mission_briefs WHERE day = '2026-10-28' AND account_id = $1`, ada).Scan(&kept))
	assert.JSONEq(t, `[{"id": "fr"}]`, kept)
	var briefs, progress int
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT (SELECT count(*) FROM mission_briefs), (SELECT count(*) FROM mission_progress)`).Scan(&briefs, &progress))
	assert.Equal(t, 1, briefs, "Germany's brief has no player")
	assert.Equal(t, 1, progress, "the progress stays")
}
