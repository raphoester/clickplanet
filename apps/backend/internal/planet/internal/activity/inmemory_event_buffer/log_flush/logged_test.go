package log_flush_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/inmemory_event_buffer"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/inmemory_event_buffer/log_flush"
)

type stubFlusher struct {
	flushed inmemory_event_buffer.Flushed
	err     error
}

func (s stubFlusher) Flush(context.Context) (inmemory_event_buffer.Flushed, error) {
	return s.flushed, s.err
}

func run(t *testing.T, inner stubFlusher) (string, inmemory_event_buffer.Flushed, error) {
	t.Helper()

	var logs bytes.Buffer
	flushed, err := log_flush.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Flush(t.Context())
	return logs.String(), flushed, err
}

func TestAFlushThatWentWellSaysNothing(t *testing.T) {
	logs, flushed, err := run(t, stubFlusher{flushed: inmemory_event_buffer.Flushed{Written: 40}})

	require.NoError(t, err)
	assert.Equal(t, 40, flushed.Written, "the answer is the inner one")
	assert.Empty(t, logs)
}

func TestAFailedFlushIsAnError(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, _, err := run(t, stubFlusher{flushed: inmemory_event_buffer.Flushed{Dropped: 3}, err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=ERROR msg="failed to flush the activity, retrying next tick" dropped=3`)
}

func TestADropIsAWarning(t *testing.T) {
	logs, _, err := run(t, stubFlusher{flushed: inmemory_event_buffer.Flushed{Written: 2, Dropped: 5}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="the activity buffer was full, events dropped" dropped=5`)
}
