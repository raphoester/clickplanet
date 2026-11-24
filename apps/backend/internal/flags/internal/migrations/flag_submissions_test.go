package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

const (
	soldFlags       = 20261123163738
	flagSubmissions = 20261124083632
)

func migratedTo(t *testing.T, version uint) *cppg.Postgres {
	t.Helper()
	db := cppg.New(cppgtest.StartServer(t).ConfigFor("flags"))
	require.NoError(t, db.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.MigrateTo(t.Context(), migrations.FS, version))
	return db
}

func seed(t *testing.T, db *cppg.Postgres, id, name string) {
	t.Helper()
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO flags (id, name, state, decided_at, ends_at, picture_hash, picture_width, picture_height,
			picture_stretch, picture_focus_x, picture_focus_y)
		VALUES ($1, $2, 'open', now(), now() + interval '30 days', repeat('ab', 32), 768, 461, false, 0.5, 0.5)`,
		id, name)
	require.NoError(t, err)
}

// Production held two flags of Corsica when the pins came: each gets Ajaccio's, and Brittany Rennes'.
func TestTheFlagsOpenedBeforeThePinsGetTheirsByName(t *testing.T) {
	db := migratedTo(t, soldFlags)
	seed(t, db, "r-6k0qh1902z", "corsica")
	seed(t, db, "r-vscrt0jglu", "Corsica")
	seed(t, db, "r-0000000001", "Brittany")

	require.NoError(t, db.MigrateTo(t.Context(), migrations.FS, flagSubmissions))

	rows, err := db.On(t.Context()).QueryContext(t.Context(),
		`SELECT id, pin_latitude, pin_longitude FROM flags ORDER BY id`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	pins := map[string][2]float64{}
	for rows.Next() {
		var id string
		var pin [2]float64
		require.NoError(t, rows.Scan(&id, &pin[0], &pin[1]))
		pins[id] = pin
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, map[string][2]float64{
		"r-6k0qh1902z": {41.9192, 8.7386}, "r-vscrt0jglu": {41.9192, 8.7386}, "r-0000000001": {48.1173, -1.6778},
		"xb": {48.1173, -1.6778},
	}, pins, "Brittany left the build in the migration before")
}

func TestAFlagWithNoKnownPinStopsTheMigration(t *testing.T) {
	db := migratedTo(t, soldFlags)
	seed(t, db, "r-0000000001", "Savoy")

	err := db.MigrateTo(t.Context(), migrations.FS, flagSubmissions)

	require.ErrorContains(t, err, "has no pin")
}
