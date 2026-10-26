package log_play_schedule_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/usecases/play_schedule_usecase/log_play_schedule"
)

type stubExecutor struct {
	turn schedule.Turn
	told bool
	err  error
}

func (s stubExecutor) Execute(context.Context) (schedule.Turn, bool, error) {
	return s.turn, s.told, s.err
}

func defaultTurn(t *testing.T) schedule.Turn {
	t.Helper()

	return schedule.New().TurnAt(time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC))
}

func run(ctx context.Context, inner stubExecutor) (string, error) {
	var logs bytes.Buffer
	_, _, err := log_play_schedule.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), err
}

func TestATurnToldIsLoggedWithItsSet(t *testing.T) {
	logs, err := run(t.Context(), stubExecutor{turn: defaultTurn(t), told: true})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="the schedule plays" set=default`)
}

func TestATurnThatHoldsSaysNothing(t *testing.T) {
	logs, err := run(t.Context(), stubExecutor{turn: defaultTurn(t)})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestAFailureIsLoggedButNotOneTheShutdownCut(t *testing.T) {
	down := errors.New("redis is down")

	logs, err := run(t.Context(), stubExecutor{err: down})
	require.ErrorIs(t, err, down)
	assert.Contains(t, logs, `level=ERROR msg="failed to play the schedule" error="redis is down"`)

	cut, cancel := context.WithCancel(t.Context())
	cancel()
	logs, err = run(cut, stubExecutor{err: down})
	require.ErrorIs(t, err, down)
	assert.Empty(t, logs)
}
