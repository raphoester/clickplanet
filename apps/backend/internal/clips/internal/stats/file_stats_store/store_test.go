package file_stats_store_test

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/file_stats_store"
)

var (
	monday  = time.Date(2026, 10, 26, 5, 41, 7, 0, time.UTC)
	tuesday = monday.Add(24 * time.Hour)
)

// A key order of its own, a number written in a way Go would not and a metric no code here knows: what is kept is what
// came, its spaces aside (a row is one line).
const entry = `{"platform":"tiktok","analytics":{"views":120,"completionRate":0.310,"aNewMetric":{"x":1}}}`

func rowAt(takenAt time.Time, post stats.PostID) stats.Row {
	return stats.Row{
		Version: stats.RowVersion, TakenAt: takenAt, PostID: post, Platform: "tiktok", PublishedAt: monday.Add(-time.Hour),
		AgeHours: 1, Answer: stats.Answer{HTTPStatus: 200, Status: "published", SyncStatus: "synced"},
		Analytics: json.RawMessage(entry), Metadata: json.RawMessage(`{"storyFlag":"fr"}`),
		Retention: &stats.Retention{HTTPStatus: 412, Body: json.RawMessage(`{"code":"youtube_analytics_scope_missing"}`)},
	}
}

func readTable(t *testing.T, folder string) [][]string {
	t.Helper()

	file, err := os.Open(filepath.Join(folder, "clips-stats.csv")) //nolint:gosec // the file the store wrote in the test's folder.
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	records, err := csv.NewReader(file).ReadAll()
	require.NoError(t, err)
	return records
}

func TestASnapshotIsAFileOfRowsKeptAsTheyCameAndTheTableIsMadeOfEverySnapshot(t *testing.T) {
	folder := t.TempDir()
	store := file_stats_store.Store(folder)

	require.NoError(t, store.Keep(t.Context(), monday, []stats.Row{rowAt(monday, "post-1")}))
	require.NoError(t, store.Keep(t.Context(), tuesday, []stats.Row{rowAt(tuesday, "post-1"), rowAt(tuesday, "post-2")}))

	files, err := filepath.Glob(filepath.Join(folder, "snapshots", "*"))
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(folder, "snapshots", "2026-10-26T05-41-07Z.jsonl"),
		filepath.Join(folder, "snapshots", "2026-10-27T05-41-07Z.jsonl"),
	}, files)

	rows, err := store.Rows()
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, rowAt(monday, "post-1"), rows[0])
	assert.JSONEq(t, entry, string(rows[0].Analytics))
	assert.Contains(t, string(rows[0].Analytics), "0.310", "a number as it was written")
	assert.Equal(t, stats.PostID("post-2"), rows[2].PostID)

	records := readTable(t, folder)
	require.Len(t, records, 4, "a header and every row of every snapshot")
	assert.Equal(t, "takenAt", records[0][0])
	assert.Equal(t, "2026-10-26T05:41:07Z", records[1][0])
	assert.Equal(t, "2026-10-27T05:41:07Z", records[3][0])
	assert.Contains(t, records[0], "metrics.aNewMetric.x")
	assert.Contains(t, records[1], "0.310")
}

func TestASnapshotOfNothingWritesNoFileAndStillMakesTheTable(t *testing.T) {
	folder := t.TempDir()

	require.NoError(t, file_stats_store.Store(folder).Keep(t.Context(), monday, nil))

	files, err := filepath.Glob(filepath.Join(folder, "snapshots", "*"))
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Len(t, readTable(t, folder), 1)
}

func TestALineThatIsNoRowFailsTheTableAndSaysWhere(t *testing.T) {
	folder := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(folder, "snapshots"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(folder, "snapshots", "2026-10-25T05-41-07Z.jsonl"), []byte("{}\nnot json\n"), 0o600))

	err := file_stats_store.Store(folder).Keep(t.Context(), monday, []stats.Row{rowAt(monday, "post-1")})

	require.ErrorContains(t, err, "line 2 of the snapshot 2026-10-25T05-41-07Z.jsonl is not a row")
}
