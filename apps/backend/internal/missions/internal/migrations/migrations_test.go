package migrations_test

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestTheOutboxKeepsEveryMessageMissionsSends(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "missions", migrations.FS)

	require.NoError(t, db.InTx(t.Context(), func(ctx context.Context) error {
		return cpoutbox.New(db).Record(ctx, &planetv1.GrantCharges{}, &scoringv1.GrantPoints{}, &missionsv1.MissionsPaid{},
			&missionsv1.MissionsDone{}, &missionsv1.FlagMissionDone{})
	}))
}

// The migrations before the one named, as a database that never ran it holds them.
func before(t *testing.T, version string) fs.FS {
	t.Helper()

	older := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	require.NoError(t, err)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") && entry.Name() < version {
			content, err := fs.ReadFile(migrations.FS, entry.Name())
			require.NoError(t, err)
			older[entry.Name()] = &fstest.MapFile{Data: content}
		}
	}
	return older
}

func TestEveryDayKeptBeforeTheBooksTakesItsSeasonsBook(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "missions", before(t, "20261116211335"))
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO mission_days (day_id, season, number, starts_at, ends_at) VALUES
			('season-0/day-24', 0, 24, '2026-10-30 21:00Z', '2026-10-31 21:00Z'),
			('season-1/day-16', 1, 16, '2026-11-15 21:00Z', '2026-11-16 21:00Z')
	`)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	books := map[string]string{}
	rows, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT day_id, book_id FROM mission_days`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var day, book string
		require.NoError(t, rows.Scan(&day, &book))
		books[day] = book
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, map[string]string{"season-0/day-24": "season-0", "season-1/day-16": "season-1"}, books)
}
