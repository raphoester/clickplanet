package log_post_teasers_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/usecases/post_teasers_usecase/log_post_teasers"
)

type stub teaser_job.Report

func (s stub) Execute(context.Context) (teaser_job.Report, error) {
	return teaser_job.Report(s), nil
}

func TestEachOutcomeIsLoggedOnEachPlatformWithWhy(t *testing.T) {
	at := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	var logs bytes.Buffer

	_, err := log_post_teasers.New(stub{
		{Battle: "entry-7", Result: teaser_job.Posted, At: at, Post: "post-1", Platforms: []string{"tiktok", "youtube"}, Refused: map[string]string{"youtube": "no account"}},
		{Battle: "entry-8", Result: teaser_job.Failed, Why: "refused"},
		{Battle: "entry-9", Result: teaser_job.Held, Why: "no room"},
	}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())

	require.NoError(t, err)
	assert.Contains(t, logs.String(), `level=INFO msg="scheduled a teaser" battle=entry-7 platform=tiktok at=2026-11-13T14:00:00.000Z post=post-1`)
	assert.Contains(t, logs.String(), `level=ERROR msg="a platform refused a teaser" battle=entry-7 platform=youtube at=2026-11-13T14:00:00.000Z why="no account"`)
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to post a teaser" battle=entry-8 why=refused`)
	assert.Contains(t, logs.String(), `level=WARN msg="a teaser was held" battle=entry-9 why="no room"`)
}
