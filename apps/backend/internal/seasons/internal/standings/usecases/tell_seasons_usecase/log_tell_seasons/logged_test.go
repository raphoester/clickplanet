package log_tell_seasons_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/tell_seasons_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/tell_seasons_usecase/log_tell_seasons"
)

type stubExecutor struct {
	told []tell_seasons_usecase.Told
	err  error
}

func (s stubExecutor) Execute(context.Context, []calendar.Number) ([]tell_seasons_usecase.Told, error) {
	return s.told, s.err
}

func run(ctx context.Context, inner stubExecutor) (string, []tell_seasons_usecase.Told, error) {
	var logs bytes.Buffer
	told, err := log_tell_seasons.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx, nil)
	return logs.String(), told, err
}

func TestATickThatToldNothingLogsNothing(t *testing.T) {
	logs, _, err := run(t.Context(), stubExecutor{})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestEachSeasonToldIsLoggedAndPassedOn(t *testing.T) {
	logs, told, err := run(t.Context(), stubExecutor{told: []tell_seasons_usecase.Told{{Season: 0, Players: 1234}}})

	require.NoError(t, err)
	assert.Equal(t, []tell_seasons_usecase.Told{{Season: 0, Players: 1234}}, told)
	assert.Contains(t, logs, "season=0")
	assert.Contains(t, logs, "players=1234")
}

func TestAFailureIsLoggedUnlessTheShutdownCutIt(t *testing.T) {
	failure := errors.New("postgres is down")

	logs, _, err := run(t.Context(), stubExecutor{err: failure})
	require.ErrorIs(t, err, failure)
	assert.Contains(t, logs, "postgres is down")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	logs, _, _ = run(ctx, stubExecutor{err: failure})
	assert.Empty(t, logs)
}
