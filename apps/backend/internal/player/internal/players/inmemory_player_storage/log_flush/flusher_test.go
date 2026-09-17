package log_flush_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage/log_flush"
)

type flusher struct {
	err error
}

func (f flusher) Flush(context.Context) error { return f.err }

func TestAFailedFlushIsLoggedAndPassedOn(t *testing.T) {
	var out bytes.Buffer
	failure := errors.New("postgres is down")

	err := log_flush.New(flusher{err: failure}, slog.New(slog.NewTextHandler(&out, nil))).Flush(t.Context())

	assert.ErrorIs(t, err, failure)
	assert.Contains(t, out.String(), "failed to flush the players")
}

func TestAFlushThatWorkedLogsNothing(t *testing.T) {
	var out bytes.Buffer

	err := log_flush.New(flusher{}, slog.New(slog.NewTextHandler(&out, nil))).Flush(t.Context())

	require.NoError(t, err)
	assert.Empty(t, out.String())
}
