package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const (
	timeRefused = 20261124171429
	flagHolders = 20261126153621
)

func pinned(t *testing.T, db *cppg.Postgres, id, name string) {
	t.Helper()
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO flags (id, name, state, decided_at, ends_at, picture_hash, picture_width, picture_height,
			picture_stretch, picture_focus_x, picture_focus_y, pin_latitude, pin_longitude)
		VALUES ($1, $2, 'open', now(), now() + interval '30 days', repeat('ab', 32), 768, 461, false, 0.5, 0.5, 48.1,
			11.5)`, id, name)
	require.NoError(t, err)
}

// Production's flags when holders came: a flag opened from a submission, one the operator closed with a note, and its
// test Bavaria, closed before the notes existed.
func TestEachFlagIsHeldByItsFounderAndEachOneClosedWithANoteIsRemoved(t *testing.T) {
	db := migratedTo(t, timeRefused)
	pinned(t, db, "r-0000000001", "Sardinia")
	pinned(t, db, "r-0000000002", "Savoy")
	pinned(t, db, "r-0000000003", "Bavaria")
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		UPDATE flags SET state = 'closed' WHERE id IN ('r-0000000002', 'r-0000000003');
		INSERT INTO closures (flag, note) VALUES ('r-0000000002', 'Broke the rules.');
		INSERT INTO submissions (id, account_id, flag, name, pin_latitude, pin_longitude, picture_hash, picture_width,
			picture_height, picture_stretch, picture_focus_x, picture_focus_y, picture_other_colors, picture_held, state,
			submitted_at, decided_at)
		VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-00000000000a', 'r-0000000001',
			'Sardinia', 40.1, 9.0, repeat('ab', 32), 300, 200, false, 0.5, 0.5, 3, false, 'opened', now(), now())`)
	require.NoError(t, err)

	require.NoError(t, db.MigrateTo(t.Context(), migrations.FS, flagHolders))

	held := map[string]string{}
	rows, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT flag, account_id::text FROM holders`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var flag, account string
		require.NoError(t, rows.Scan(&flag, &account))
		held[flag] = account
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, map[string]string{"r-0000000001": "00000000-0000-0000-0000-00000000000a"}, held,
		"the flags opened by hand (Brittany, England, Scotland, Wales) have none")

	states := map[string]string{}
	read, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT id, state::text FROM flags WHERE id LIKE 'r-%'`)
	require.NoError(t, err)
	defer func() { _ = read.Close() }()
	for read.Next() {
		var id, state string
		require.NoError(t, read.Scan(&id, &state))
		states[id] = state
	}
	require.NoError(t, read.Err())
	assert.Equal(t, map[string]string{"r-0000000001": "open", "r-0000000002": "removed", "r-0000000003": "closed"},
		states)
}

func TestARemovedFlagsNameIsFreeAndAClosedOnesIsNot(t *testing.T) {
	db := migratedTo(t, timeRefused)
	pinned(t, db, "r-0000000001", "Savoy")
	pinned(t, db, "r-0000000002", "Bavaria")
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		UPDATE flags SET state = 'closed' WHERE id IN ('r-0000000001', 'r-0000000002');
		INSERT INTO closures (flag, note) VALUES ('r-0000000001', 'Broke the rules.')`)
	require.NoError(t, err)
	require.NoError(t, db.MigrateTo(t.Context(), migrations.FS, flagHolders))

	pinned(t, db, "r-0000000003", "SAVOY")

	_, err = db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO flags (id, name, state, decided_at, ends_at, picture_hash, picture_width, picture_height,
			picture_stretch, picture_focus_x, picture_focus_y, pin_latitude, pin_longitude)
		VALUES ('r-0000000004', 'bavaria', 'open', now(), now() + interval '30 days', repeat('ab', 32), 768, 461, false,
			0.5, 0.5, 48.1, 11.5)`)
	require.ErrorContains(t, err, "flags_name_key", "a flag whose time ended keeps its name")
}
