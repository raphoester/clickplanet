package log_fetch_replay_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_replay_usecase/log_fetch_replay"
)

type stubExecutor struct {
	replay replays.Replay
	err    error
}

func (s stubExecutor) Execute(context.Context) (replays.Replay, error) {
	return s.replay, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_fetch_replay.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestASavedReplayIsLoggedWithItsEventsAndItsWindow(t *testing.T) {
	until := time.Date(2026, 10, 12, 12, 17, 0, 0, time.UTC)

	logs, err := run(t, stubExecutor{replay: replays.Replay{
		Document: []byte("{}"), Events: 4821, Window: replays.Window{Since: until.Add(-12 * time.Hour), Until: until},
	}})

	require.NoError(t, err)
	assert.Contains(t, logs, `msg="saved a replay" events=4821 since=2026-10-12T00:17:00.000Z until=2026-10-12T12:17:00.000Z`)
	assert.NotContains(t, logs, "{}")
}

func TestAFailedFetchSaysNothing(t *testing.T) {
	logs, err := run(t, stubExecutor{err: errors.New("ssh failed")})

	require.Error(t, err)
	assert.Empty(t, logs)
}
