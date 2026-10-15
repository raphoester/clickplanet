package log_history_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/log_history"
)

var at = time.Date(2026, 10, 15, 7, 15, 0, 0, time.UTC)

type history struct {
	told []posts.Telling
	err  error
}

func (h history) Told(context.Context, time.Time) ([]posts.Telling, error) {
	return h.told, h.err
}

func read(t *testing.T, inner history) (string, []posts.Telling, error) {
	t.Helper()

	var logs bytes.Buffer
	told, err := log_history.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Told(t.Context(), at.Add(-24*time.Hour))
	return logs.String(), told, err
}

func TestAPostThatWentOutIsLoggedWithItsState(t *testing.T) {
	telling := posts.Telling{Key: posts.StoryKey{Flag: "dz", Continent: "Africa"}, At: at, Title: "ALGERIA IS INVADING LIBYA", State: "published"}

	logs, told, err := read(t, history{told: []posts.Telling{telling}})

	require.NoError(t, err)
	assert.Equal(t, []posts.Telling{telling}, told)
	assert.Contains(t, logs, `level=INFO msg="a post of the last day" clip="ALGERIA IS INVADING LIBYA" at=2026-10-15T07:15:00.000Z state=published`)
}

func TestAPostAPlatformRefusedIsAWarningThatSaysWhy(t *testing.T) {
	logs, _, err := read(t, history{told: []posts.Telling{{
		Key: posts.StoryKey{Flag: "fr", Continent: "Africa"}, At: at, Title: "FRANCE IS KICKING MOROCCO OUT OF LIBYA", State: "failed",
		Failure: "tiktok: TikTok creator has reached the daily posting limit.",
	}}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="a post of the last day was refused" clip="FRANCE IS KICKING MOROCCO OUT OF LIBYA"`)
	assert.Contains(t, logs, `failure="tiktok: TikTok creator has reached the daily posting limit."`)
}

func TestAHistoryThatCannotBeReadSaysNothing(t *testing.T) {
	unreachable := errors.New("zernio could not be reached")

	logs, _, err := read(t, history{err: unreachable})

	require.ErrorIs(t, err, unreachable)
	assert.Empty(t, logs)
}
