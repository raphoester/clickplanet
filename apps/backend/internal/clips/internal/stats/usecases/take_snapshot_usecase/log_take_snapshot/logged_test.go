package log_take_snapshot_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/usecases/take_snapshot_usecase/log_take_snapshot"
)

type stubExecutor struct {
	report stats.Report
	err    error
}

func (s stubExecutor) Execute(context.Context) (stats.Report, error) {
	return s.report, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_take_snapshot.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestASnapshotLogsItsCountsEachTikTokLaneAndEachPostLost(t *testing.T) {
	logs, err := run(t, stubExecutor{report: stats.Report{
		Posts: 5, NotOut: 1, Rows: map[string]int{"tiktok": 4, "youtube": 3}, Syncing: 2,
		TikTokAccounts: []stats.TikTokLane{{TikTokAccount: stats.TikTokAccount{Username: "clickplanet", Type: "business", CommercialMusic: true}, BusinessRows: 2}},
		Unread:         []stats.Unread{{PostID: "post-9", Reason: "gone"}},
		CurvesRefused:  map[string]int{"412 youtube_analytics_scope_missing": 3},
	}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="a post of the job could not be read whole, and its snapshot is lost" post=post-9 reason=gone`)
	assert.Contains(t, logs, `msg="a tiktok account" username=clickplanet type=business commercialMusic=true businessRows=2 businessLaneLive=true`)
	assert.Contains(t, logs, `msg="took a snapshot of the posts of the job" posts=5 notOut=1 rows=7 syncing=2`)
	assert.Contains(t, logs, `level=WARN msg="youtube gave no retention curve" why="412 youtube_analytics_scope_missing: connect the YouTube account again in Zernio to grant the analytics scope" videos=3`)
}

func TestAFailedSnapshotLogsNothing(t *testing.T) {
	logs, err := run(t, stubExecutor{err: errors.New("no")})

	require.EqualError(t, err, "no")
	assert.Empty(t, logs)
}
