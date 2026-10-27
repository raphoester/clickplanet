package log_close_ended_rounds_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/close_ended_rounds_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/close_ended_rounds_usecase/log_close_ended_rounds"
)

type stubExecutor struct {
	report close_ended_rounds_usecase.Report
	err    error
}

func (s stubExecutor) Execute(context.Context) (close_ended_rounds_usecase.Report, error) {
	return s.report, s.err
}

func run(ctx context.Context, inner stubExecutor) (string, close_ended_rounds_usecase.Report, error) {
	var logs bytes.Buffer
	report, err := log_close_ended_rounds.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), report, err
}

var finale = rounds.Closed{
	Round:   rounds.Round{EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), Finale: true},
	Number:  23,
	Results: []rounds.Result{{Country: "fr", Rank: 1, Points: 75, Share: 0.3}},
}

func TestACallThatClosedNothingLogsNothing(t *testing.T) {
	logs, _, err := run(t.Context(), stubExecutor{})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestARoundClosedAtItsEndIsLoggedAndPassedOn(t *testing.T) {
	report := close_ended_rounds_usecase.Report{
		Closed: []rounds.Closed{finale},
		ReadAt: finale.Round.EndsAt.Add(20 * time.Millisecond),
	}

	logs, passed, err := run(t.Context(), stubExecutor{report: report})

	require.NoError(t, err)
	assert.Equal(t, report, passed)
	assert.Contains(t, logs, `level=INFO msg="closed a round" season=0 number=23 endsAt=2026-10-31T23:00:00.000Z finale=true ranked=1 late=20ms`)
}

func TestARoundClosedFromAMapReadLateIsAWarning(t *testing.T) {
	report := close_ended_rounds_usecase.Report{
		Closed: []rounds.Closed{finale},
		ReadAt: finale.Round.EndsAt.Add(3 * time.Minute),
	}

	logs, _, err := run(t.Context(), stubExecutor{report: report})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="closed a round late, from the map as read then" season=0 number=23`)
	assert.Contains(t, logs, `late=3m0s`)
}

func TestAFailureIsLoggedAndPassedOn(t *testing.T) {
	failure := errors.New("planet is down")

	logs, _, err := run(t.Context(), stubExecutor{err: failure})

	require.ErrorIs(t, err, failure)
	assert.Contains(t, logs, `level=ERROR msg="failed to close the rounds that ended" error="planet is down"`)
}

func TestAFailureWhileStoppingIsNotLogged(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	logs, _, err := run(ctx, stubExecutor{err: context.Canceled})

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, logs)
}
