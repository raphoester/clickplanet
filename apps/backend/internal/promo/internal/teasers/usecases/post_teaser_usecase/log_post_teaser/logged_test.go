package log_post_teaser_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teaser_usecase/log_post_teaser"
)

type stub teasers.Outcome

func (s stub) Execute(context.Context, teasers.BattleID, teasers.Media) (teasers.Outcome, error) {
	return teasers.Outcome(s), nil
}

func run(t *testing.T, outcome teasers.Outcome) string {
	t.Helper()
	var logs bytes.Buffer
	_, err := log_post_teaser.New(stub(outcome), slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), "entry-7", teasers.Media{})
	require.NoError(t, err)
	return logs.String()
}

func TestEachOutcomeIsLoggedOnEachPlatformWithWhy(t *testing.T) {
	at := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)

	logs := run(t, teasers.Outcome{
		Result: teasers.Posted, At: at, Post: "post-1", Platforms: []teasers.Platform{teasers.TikTok, teasers.YouTube},
		Refused: map[teasers.Platform]string{teasers.YouTube: "no account"},
	})
	assert.Contains(t, logs, `level=INFO msg="scheduled a teaser" platform=tiktok at=2026-11-13T14:00:00.000Z battle=entry-7 post=post-1`)
	assert.Contains(t, logs, `level=ERROR msg="a platform refused a teaser" platform=youtube at=2026-11-13T14:00:00.000Z battle=entry-7 why="no account"`)

	assert.Contains(t, run(t, teasers.Outcome{Result: teasers.Failed, Why: "refused"}), `level=ERROR msg="a teaser was refused" battle=entry-7 why=refused`)
	assert.Contains(t, run(t, teasers.Outcome{Result: teasers.Held, Why: "no room"}), `level=WARN msg="a teaser was held" battle=entry-7 why="no room"`)
}
