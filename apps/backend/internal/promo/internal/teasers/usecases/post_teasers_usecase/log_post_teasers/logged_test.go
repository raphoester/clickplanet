package log_post_teasers_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teasers_usecase/log_post_teasers"
)

type stub struct {
	report teasers.Report
	err    error
}

func (s stub) Execute(context.Context) (teasers.Report, error) {
	return s.report, s.err
}

func TestEachOutcomeIsLoggedOnEachPlatformWithWhy(t *testing.T) {
	at := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	var logs bytes.Buffer
	failure := errors.New("ssh failed")

	_, err := log_post_teasers.New(stub{err: failure, report: teasers.Report{
		{
			Battle: "entry-7", Title: "BATTLE", Result: teasers.Scheduled, At: at, Post: "post-1",
			Platforms: []teasers.Platform{teasers.TikTok, teasers.YouTube}, Refused: map[teasers.Platform]string{teasers.YouTube: "no account"},
		},
		{Battle: "entry-8", Title: "BATTLE", Result: teasers.Failed, Why: "refused"},
		{Battle: "entry-9", Title: "BATTLE", Result: teasers.Held, Why: "no platform takes it"},
	}}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())

	require.ErrorIs(t, err, failure)
	assert.Contains(t, logs.String(), `level=INFO msg="scheduled a teaser" battle=entry-7 teaser=BATTLE platform=tiktok at=2026-11-13T14:00:00.000Z post=post-1`)
	assert.Contains(t, logs.String(), `level=ERROR msg="a platform refused a teaser" battle=entry-7 teaser=BATTLE platform=youtube at=2026-11-13T14:00:00.000Z why="no account"`)
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to schedule a teaser" battle=entry-8 teaser=BATTLE why=refused`)
	assert.Contains(t, logs.String(), `level=WARN msg="a teaser was held" battle=entry-9 teaser=BATTLE why="no platform takes it"`)
}
