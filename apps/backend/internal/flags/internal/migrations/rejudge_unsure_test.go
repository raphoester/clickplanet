package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
)

const rejudgeUnsure = 20261127111140

// Production's judgements when the rule grew: Solandia left unsure as it waits, a flag unsure then rejected by the
// operator, one approved and one rejected by the judge.
func TestOnlyAnUnsureJudgementOfOneThatWaitsIsDeleted(t *testing.T) {
	db := migratedTo(t, flagHolders)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO submissions (id, account_id, flag, name, pin_latitude, pin_longitude, picture_hash, picture_width,
			picture_height, picture_stretch, picture_focus_x, picture_focus_y, picture_other_colors, picture_held, state,
			submitted_at, decided_at)
		VALUES
			('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-00000000000a', '', 'Solandia', 10, 10,
				repeat('ab', 32), 300, 200, false, 0.5, 0.5, 3, true, 'waiting', now(), now()),
			('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-00000000000b', '', 'Savoy', 45.5, 6.4,
				repeat('ab', 32), 300, 200, false, 0.5, 0.5, 3, true, 'rejected', now(), now()),
			('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-00000000000c', '', 'Tyrol', 47.2, 11.4,
				repeat('ab', 32), 300, 200, false, 0.5, 0.5, 3, false, 'approved', now(), now()),
			('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-00000000000d', '', 'Smurfland', 0, 0,
				repeat('ab', 32), 300, 200, false, 0.5, 0.5, 3, true, 'rejected', now(), now());
		INSERT INTO rejections (submission_id, reason, note) VALUES
			('00000000-0000-0000-0000-000000000002', 'other', 'Not now.'),
			('00000000-0000-0000-0000-000000000004', 'not_a_place', 'A cartoon.');
		INSERT INTO judgements (submission_id, verdict, why, judged_at) VALUES
			('00000000-0000-0000-0000-000000000001', 'unsure', 'could be fictional', now()),
			('00000000-0000-0000-0000-000000000002', 'unsure', 'not sure', now()),
			('00000000-0000-0000-0000-000000000003', 'approve', 'Tyrol''s eagle', now()),
			('00000000-0000-0000-0000-000000000004', 'reject', 'a cartoon', now())`)
	require.NoError(t, err)

	require.NoError(t, db.MigrateTo(t.Context(), migrations.FS, rejudgeUnsure))

	kept := map[string]string{}
	rows, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT submission_id::text, verdict::text FROM judgements`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var submission, verdict string
		require.NoError(t, rows.Scan(&submission, &verdict))
		kept[submission] = verdict
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, map[string]string{
		"00000000-0000-0000-0000-000000000002": "unsure",
		"00000000-0000-0000-0000-000000000003": "approve",
		"00000000-0000-0000-0000-000000000004": "reject",
	}, kept, "Solandia's goes, so the judge reads it again")
}
