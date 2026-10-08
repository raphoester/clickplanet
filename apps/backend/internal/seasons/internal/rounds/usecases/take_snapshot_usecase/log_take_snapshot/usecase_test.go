package log_take_snapshot_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/take_snapshot_usecase/log_take_snapshot"
)

type stubExecutor struct {
	closed []rounds.Closed
	err    error
}

func (s stubExecutor) Execute(context.Context) ([]rounds.Closed, error) {
	return s.closed, s.err
}

func run(ctx context.Context, inner stubExecutor) (string, []rounds.Closed, error) {
	var logs bytes.Buffer
	closed, err := log_take_snapshot.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), closed, err
}

func TestASnapshotThatClosedNothingLogsNothing(t *testing.T) {
	logs, _, err := run(t.Context(), stubExecutor{})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestEachRoundClosedIsLoggedAndPassedOn(t *testing.T) {
	finale := rounds.Closed{
		Round:   rounds.Round{EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), Finale: true},
		Number:  23,
		Results: []rounds.Result{{Country: "fr", Rank: 1, Points: 75}},
	}

	logs, closed, err := run(t.Context(), stubExecutor{closed: []rounds.Closed{finale}})

	require.NoError(t, err)
	assert.Equal(t, []rounds.Closed{finale}, closed)
	assert.Contains(t, logs, `level=INFO msg="closed a round" season=0 number=23 endsAt=2026-10-31T23:00:00.000Z finale=true ranked=1`)
}

func TestAFailureIsLoggedAndPassedOn(t *testing.T) {
	failure := errors.New("planet is down")

	logs, _, err := run(t.Context(), stubExecutor{err: failure})

	require.ErrorIs(t, err, failure)
	assert.Contains(t, logs, `level=ERROR msg="failed to take the snapshot" error="planet is down"`)
}

func TestAFailureWhileStoppingIsNotLogged(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	logs, _, err := run(ctx, stubExecutor{err: context.Canceled})

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, logs)
}
