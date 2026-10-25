package ssh_stats_store_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/ssh_stats_store"
)

var takenAt = time.Date(2026, 10, 25, 5, 41, 7, 0, time.UTC)

type fakeSSH struct {
	client string
	folder string
}

// Call n's arguments, one a line, and what came on its standard input.
func (f fakeSSH) call(t *testing.T, n int) ([]string, *clipsv1.KeepSnapshotRequest) {
	t.Helper()

	args, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("args-%d", n)))
	require.NoError(t, err)
	body, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("body-%d", n)))
	require.NoError(t, err)
	var request clipsv1.KeepSnapshotRequest
	require.NoError(t, protojson.Unmarshal(body, &request))
	return strings.Split(strings.TrimSuffix(string(args), "\n"), "\n"), &request
}

func (f fakeSSH) calls(t *testing.T) int {
	t.Helper()

	bodies, err := filepath.Glob(filepath.Join(f.folder, "body-*"))
	require.NoError(t, err)
	return len(bodies)
}

// An ssh that keeps its arguments and its standard input, then says what call n is told to (says-n), and exits as told.
func ssh(t *testing.T, says []string, complains string, exit int) fakeSSH {
	t.Helper()

	folder := t.TempDir()
	for i, said := range says {
		require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("says-%d", i+1)), []byte(said), 0o600))
	}
	fake := fakeSSH{client: filepath.Join(folder, "ssh"), folder: folder}
	script := fmt.Sprintf(`#!/bin/sh
n=$(( $(ls %[1]q | grep -c '^body-') + 1 ))
for arg in "$@"; do printf '%%s\n' "$arg"; done > %[1]q/args-$n
cat > %[1]q/body-$n
cat %[1]q/says-$n 2> /dev/null
printf '%%s' %[2]q >&2
exit %[3]d
`, folder, complains, exit)
	require.NoError(t, os.WriteFile(fake.client, []byte(script), 0o700)) //nolint:gosec // a script the test runs.
	return fake
}

func rows(n int) []stats.Row {
	made := make([]stats.Row, 0, n)
	for i := range n {
		made = append(made, stats.Row{
			Version: stats.RowVersion, TakenAt: takenAt, PostID: stats.PostID(fmt.Sprintf("post-%d", i)), Platform: "tiktok",
			Answer: stats.Answer{HTTPStatus: 200}, Analytics: json.RawMessage(`{"analytics":{"completionRate":0.310}}`),
			Metadata: json.RawMessage(`{"storyFlag":"fr"}`),
		})
	}
	return made
}

func aReport(kept int) stats.Report {
	return stats.Report{
		TakenAt: takenAt, Since: takenAt.Add(-28 * 24 * time.Hour), Posts: 3, NotOut: 1, Rows: map[string]int{"tiktok": kept},
		Syncing: 1, Curves: 2,
		TikTokAccounts: []stats.TikTokLane{{
			TikTokAccount: stats.TikTokAccount{ID: "tt-1", Username: "clickplanet", Type: "business", CommercialMusic: true},
			Rows:          kept, BusinessRows: 1,
		}},
		Unread: []stats.Unread{{PostID: "post-9", Reason: "gone"}},
	}
}

func TestASnapshotIsSentToTheOperatorToolOverSSHOnItsStandardInput(t *testing.T) {
	fake := ssh(t, []string{`{"rows":2}`}, "", 0)
	store, err := ssh_stats_store.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	require.NoError(t, store.Keep(t.Context(), aReport(2), rows(2)))

	args, request := fake.call(t, 1)
	assert.Equal(t, []string{
		"--",
		"deploy@vps.example",
		"cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO- --header 'Content-Type: application/json'" +
			" --post-file=/dev/stdin http://127.0.0.1:8081/clips.v1.AdminService/KeepSnapshot",
	}, args)
	summary := request.GetSnapshot()
	assert.Equal(t, takenAt, summary.GetTakenAt().AsTime())
	assert.Equal(t, uint32(3), summary.GetPosts())
	assert.Equal(t, uint32(1), summary.GetNotOut())
	assert.Equal(t, uint32(2), summary.GetRows())
	assert.Equal(t, uint32(1), summary.GetSyncing())
	assert.Equal(t, uint32(2), summary.GetCurves())
	require.Len(t, summary.GetTiktokAccounts(), 1)
	lane := summary.GetTiktokAccounts()[0]
	assert.Equal(t, "business", lane.GetAccountType())
	assert.True(t, lane.GetLive())
	assert.Equal(t, uint32(1), lane.GetBusinessRows())
	assert.Equal(t, "post-9", summary.GetUnread()[0].GetPostId())

	require.Len(t, request.GetRows(), 2)
	row := request.GetRows()[0]
	assert.Equal(t, "post-0", row.GetPostId())
	assert.Equal(t, "tiktok", row.GetPlatform())
	assert.Equal(t, takenAt, row.GetTakenAt().AsTime())
	assert.Contains(t, row.GetDocument(), `"completionRate":0.310`, "the row as the job wrote it, numbers and all")
	var document stats.Row
	require.NoError(t, json.Unmarshal([]byte(row.GetDocument()), &document))
	assert.Equal(t, rows(1)[0].PostID, document.PostID)
}

func TestALargeSnapshotIsSentInPartsEachWithTheReport(t *testing.T) {
	fake := ssh(t, []string{`{"rows":200}`, `{"rows":200}`, `{"rows":50}`}, "", 0)
	store, err := ssh_stats_store.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	require.NoError(t, store.Keep(t.Context(), aReport(450), rows(450)))

	require.Equal(t, 3, fake.calls(t))
	for n, size := range map[int]int{1: 200, 2: 200, 3: 50} {
		_, request := fake.call(t, n)
		assert.Len(t, request.GetRows(), size)
		assert.Equal(t, uint32(450), request.GetSnapshot().GetRows(), "each part carries the whole snapshot's summary")
	}
	_, last := fake.call(t, 3)
	assert.Equal(t, "post-400", last.GetRows()[0].GetPostId())
}

func TestASnapshotOfNoRowIsStillKept(t *testing.T) {
	fake := ssh(t, []string{`{}`}, "", 0)
	store, err := ssh_stats_store.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	require.NoError(t, store.Keep(t.Context(), aReport(0), nil))

	require.Equal(t, 1, fake.calls(t))
	_, request := fake.call(t, 1)
	assert.Empty(t, request.GetRows())
}

func TestABackendThatKeptLessThanItWasSentFailsTheSnapshot(t *testing.T) {
	fake := ssh(t, []string{`{"rows":1}`}, "", 0)
	store, err := ssh_stats_store.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	err = store.Keep(t.Context(), aReport(2), rows(2))

	require.ErrorContains(t, err, "the backend kept 1 rows of 2")
}

func TestAnSSHThatFailsSaysWhatItComplainedOf(t *testing.T) {
	fake := ssh(t, nil, "Permission denied (publickey).", 255)
	store, err := ssh_stats_store.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	err = store.Keep(t.Context(), aReport(2), rows(2))

	require.ErrorContains(t, err, "failed to keep rows 1 to 2 of 2: ssh failed: exit status 255: Permission denied (publickey).")
}

func TestAnAnswerThatIsNoResponseFailsTheSnapshot(t *testing.T) {
	fake := ssh(t, []string{`<html>502</html>`}, "", 0)
	store, err := ssh_stats_store.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	err = store.Keep(t.Context(), aReport(2), rows(2))

	require.ErrorContains(t, err, "not a clips.v1.KeepSnapshotResponse")
}

func TestNoHostIsRefused(t *testing.T) {
	_, err := ssh_stats_store.New("", "ssh")

	require.ErrorContains(t, err, "the ssh host is empty")
}

// Call n's arguments, and what came on its standard input, as it came.
func (f fakeSSH) raw(t *testing.T, n int) ([]string, string) {
	t.Helper()

	args, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("args-%d", n)))
	require.NoError(t, err)
	body, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("body-%d", n)))
	require.NoError(t, err)
	return strings.Split(strings.TrimSuffix(string(args), "\n"), "\n"), string(body)
}

func documentOf(t *testing.T, row stats.Row) string {
	t.Helper()

	document, err := json.Marshal(row)
	require.NoError(t, err)
	quoted, err := json.Marshal(string(document))
	require.NoError(t, err)
	return string(quoted)
}

func TestTheLastSnapshotIsItsRowsEveryPageOfThem(t *testing.T) {
	youtube := stats.Row{
		Version: stats.RowVersion, TakenAt: takenAt, PostID: "post-1", Platform: "youtube", PlatformPostID: "abc",
		Retention: &stats.Retention{HTTPStatus: 200, Body: json.RawMessage(`{"retentionCurve":[{}]}`), ReadAt: takenAt, AgeHours: 50},
	}
	fake := ssh(t, []string{
		`{"snapshots":[{"takenAt":"2026-10-25T05:41:07Z"}]}`,
		`{"rows":[{"postId":"post-0","platform":"tiktok","document":` + documentOf(t, rows(1)[0]) + `}],"nextPage":"p2"}`,
		`{"rows":[{"postId":"post-1","platform":"youtube","document":` + documentOf(t, youtube) + `}]}`,
	}, "", 0)
	store, err := ssh_stats_store.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	last, err := store.Last(t.Context())

	require.NoError(t, err)
	require.Len(t, last, 2)
	assert.Equal(t, stats.PostID("post-0"), last[0].PostID)
	assert.True(t, last[1].Retention.Curve())
	assert.InDelta(t, 50, last[1].Retention.AgeHours, 0.001)

	args, body := fake.raw(t, 1)
	assert.True(t, strings.HasSuffix(args[2], "/clips.v1.AdminService/ListSnapshots"))
	assert.JSONEq(t, `{"limit":1}`, body)
	args, body = fake.raw(t, 2)
	assert.True(t, strings.HasSuffix(args[2], "/clips.v1.AdminService/ListRows"))
	assert.JSONEq(t, `{"since":"2026-10-25T05:41:07Z","until":"2026-10-25T05:41:08Z","limit":2000}`, body, "that snapshot alone")
	_, body = fake.raw(t, 3)
	assert.JSONEq(t, `{"since":"2026-10-25T05:41:07Z","until":"2026-10-25T05:41:08Z","limit":2000,"page":"p2"}`, body)
}

func TestBeforeTheFirstSnapshotThereIsNoLastOne(t *testing.T) {
	fake := ssh(t, []string{`{}`}, "", 0)
	store, err := ssh_stats_store.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	last, err := store.Last(t.Context())

	require.NoError(t, err)
	assert.Empty(t, last)
	assert.Equal(t, 1, fake.calls(t))
}

func TestALastRowThatIsNoRowFailsTheRead(t *testing.T) {
	fake := ssh(t, []string{
		`{"snapshots":[{"takenAt":"2026-10-25T05:41:07Z"}]}`,
		`{"rows":[{"postId":"post-0","platform":"tiktok","document":"[1]"}]}`,
	}, "", 0)
	store, err := ssh_stats_store.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	_, err = store.Last(t.Context())

	require.ErrorContains(t, err, "the row of post-0 on tiktok is not a row")
}
