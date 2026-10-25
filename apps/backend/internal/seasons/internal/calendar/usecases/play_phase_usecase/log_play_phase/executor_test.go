package log_play_phase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/play_phase_usecase/log_play_phase"
)

type stubExecutor struct {
	played calendar.Phase
	err    error
}

func (s stubExecutor) Execute(context.Context) (calendar.Phase, error) { return s.played, s.err }

func run(ctx context.Context, inner stubExecutor) (string, calendar.Phase, error) {
	var logs bytes.Buffer
	played, err := log_play_phase.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), played, err
}

func TestAPassThatChangedNothingLogsNothing(t *testing.T) {
	logs, _, err := run(t.Context(), stubExecutor{})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestAPhasePlayedIsLoggedAndPassedOn(t *testing.T) {
	finale := calendar.Phase{Season: 0, Stage: calendar.StageFinale}

	logs, played, err := run(t.Context(), stubExecutor{played: finale})

	require.NoError(t, err)
	assert.Equal(t, finale, played)
	assert.Contains(t, logs, `level=INFO msg="played a season phase" season=0 stage=finale`)
}

func TestAFailureIsLoggedButNotOneTheShutdownCut(t *testing.T) {
	cause := errors.New("planet is down")

	logs, _, err := run(t.Context(), stubExecutor{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=ERROR msg="failed to play the season phase" error="planet is down"`)

	cut, cancel := context.WithCancel(t.Context())
	cancel()
	logs, _, err = run(cut, stubExecutor{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Empty(t, logs)
}
