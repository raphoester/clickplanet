package log_schedule_clips_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/schedule_clips_usecase/log_schedule_clips"
)

var at = time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC)

type stubExecutor struct {
	report posts.Report
	err    error
}

func (s stubExecutor) Execute(context.Context) (posts.Report, error) {
	return s.report, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_schedule_clips.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestAScheduledClipIsLoggedOnEachPlatformWithItsTimeAndItsPost(t *testing.T) {
	logs, err := run(t, stubExecutor{report: posts.Report{{
		Title: "FRANCE IS ATTACKING EUROPE", At: at, Post: "post-1",
		Platforms: []posts.PlatformOutcome{{Platform: posts.TikTok}, {Platform: posts.YouTube}},
	}}})

	require.NoError(t, err)
	assert.Contains(t, logs,
		`level=INFO msg="scheduled a clip" clip="FRANCE IS ATTACKING EUROPE" platform=tiktok at=2026-10-12T16:40:00.000Z post=post-1`)
	assert.Contains(t, logs,
		`level=INFO msg="scheduled a clip" clip="FRANCE IS ATTACKING EUROPE" platform=youtube at=2026-10-12T16:40:00.000Z post=post-1`)
}

func TestAPlatformThatDidNotTakeAClipIsLoggedAsAnErrorWithWhy(t *testing.T) {
	logs, err := run(t, stubExecutor{report: posts.Report{{
		Title: "FRANCE IS ATTACKING EUROPE", At: at, Post: "post-1",
		Platforms: []posts.PlatformOutcome{{Platform: posts.TikTok}, {Platform: posts.YouTube, Failure: "no youtube account"}},
	}}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="scheduled a clip" clip="FRANCE IS ATTACKING EUROPE" platform=tiktok`)
	assert.Contains(t, logs, `level=ERROR msg="failed to schedule a clip" clip="FRANCE IS ATTACKING EUROPE" platform=youtube`)
	assert.Contains(t, logs, `failure="no youtube account"`)
}

func TestARunThatReadNoClipSaysNothing(t *testing.T) {
	logs, err := run(t, stubExecutor{err: posts.ErrInvalidClip})

	require.ErrorIs(t, err, posts.ErrInvalidClip)
	assert.Empty(t, logs)
}
