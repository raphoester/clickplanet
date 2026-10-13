package log_service_test

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/log_service"
)

func schedule(t *testing.T, inner *posts.ScriptedService) (string, posts.PostID, error) {
	t.Helper()

	clip, err := posts.ClipSnapshot{Pick: 1, Title: "FRANCE IS ATTACKING EUROPE", Caption: "C", Video: "clip-1.mp4"}.Restore()
	require.NoError(t, err)
	post := posts.Cadence{Review: 4 * time.Hour}.Posts([]posts.Clip{clip}, time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC))[0]

	var logs bytes.Buffer
	id, err := log_service.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Schedule(t.Context(), post)
	return logs.String(), id, err
}

func TestAPostTheServiceTakesSaysNothing(t *testing.T) {
	logs, id, err := schedule(t, posts.NewScriptedService())

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-FRANCE IS ATTACKING EUROPE"), id)
	assert.Empty(t, logs)
}

func TestAPostTheServiceDidNotTakeIsLoggedWithItsClipAndWhy(t *testing.T) {
	refusal := &posts.BusyError{RetryAfter: time.Minute, Reason: "rate limit exceeded"}

	logs, _, err := schedule(t, posts.NewScriptedService(refusal))

	require.ErrorIs(t, err, refusal)
	assert.Contains(t, logs, "level=WARN")
	assert.Contains(t, logs, `clip="FRANCE IS ATTACKING EUROPE"`)
	assert.Contains(t, logs, "rate limit exceeded")
}
