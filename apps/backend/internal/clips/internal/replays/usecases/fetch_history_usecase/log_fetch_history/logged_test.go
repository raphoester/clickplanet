package log_fetch_history_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_history_usecase/log_fetch_history"
)

type stubExecutor struct {
	history replays.History
	err     error
}

func (s stubExecutor) Execute(context.Context, replays.Turn, time.Time) (replays.History, error) {
	return s.history, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_fetch_history.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), replays.Turn{}, time.Time{})
	return logs.String(), err
}

func TestASavedMapIsLoggedWithItsWindowAndItsSizeButNotItsTiles(t *testing.T) {
	logs, err := run(t, stubExecutor{history: replays.History{
		Document: []byte(`{"history":{"frames":[{"owners":[{"tile":7,"countryId":"pw"}]}]}}`), Frames: 224,
		Window: replays.Window{Since: time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC), Until: time.Date(2026, 11, 10, 14, 20, 0, 0, time.UTC)},
	}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="saved the season's map" since=2026-11-01T15:00:00.000Z until=2026-11-10T14:20:00.000Z frames=224 bytes=65`)
	assert.NotContains(t, logs, "pw")
}

func TestNoSeasonIsNoWarning(t *testing.T) {
	logs, err := run(t, stubExecutor{err: fmt.Errorf("failed to ask for the season's map: %w", replays.ErrNoSeason)})

	require.ErrorIs(t, err, replays.ErrNoSeason)
	assert.Contains(t, logs, `level=INFO msg="left the season's map out: no season runs, so no saga is told"`)
}

func TestAMapLeftOutIsAWarningThatSaysWhy(t *testing.T) {
	unserved := errors.New("ssh failed: exit status 1: wget: server returned error: HTTP/1.1 500 Internal Server Error")

	logs, err := run(t, stubExecutor{err: unserved})

	require.ErrorIs(t, err, unserved)
	assert.Contains(t, logs, `level=WARN msg="left the season's map out: no saga is told"`)
	assert.Contains(t, logs, "500 Internal Server Error")
}
