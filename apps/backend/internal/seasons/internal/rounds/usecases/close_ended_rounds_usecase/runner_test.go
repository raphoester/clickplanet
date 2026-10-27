package close_ended_rounds_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/close_ended_rounds_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type scriptedExecutor struct {
	answers chan answer
	calls   chan time.Time
}

type answer struct {
	report close_ended_rounds_usecase.Report
	err    error
}

func (s scriptedExecutor) Execute(context.Context) (close_ended_rounds_usecase.Report, error) {
	s.calls <- time.Now()
	next := <-s.answers
	return next.report, next.err
}

func run(t *testing.T, clock cptime.Clock, retry time.Duration) (scriptedExecutor, context.CancelFunc, <-chan struct{}) {
	t.Helper()

	executor := scriptedExecutor{answers: make(chan answer, 10), calls: make(chan time.Time, 10)}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		close_ended_rounds_usecase.NewRunner(executor, clock, retry).Run(ctx)
		close(done)
	}()
	return executor, cancel, done
}

func stopped(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(time.Second):
		require.Fail(t, "the runner did not stop")
	}
}

func TestTheRunnerClosesAtStartThenWhenTheRoundInProgressEnds(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 16, 20, 59, 59, 0, time.UTC))
	executor, cancel, done := run(t, clock, time.Hour)

	started := <-executor.calls
	executor.answers <- answer{report: close_ended_rounds_usecase.Report{Next: clock.Now().Add(50 * time.Millisecond)}}
	again := <-executor.calls

	assert.GreaterOrEqual(t, again.Sub(started), 50*time.Millisecond)
	cancel()
	executor.answers <- answer{}
	stopped(t, done)
}

func TestAFailureIsTriedAgainSoon(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC))
	executor, cancel, done := run(t, clock, 20*time.Millisecond)

	<-executor.calls
	executor.answers <- answer{
		report: close_ended_rounds_usecase.Report{Next: clock.Now().Add(24 * time.Hour)},
		err:    errors.New("planet is down"),
	}

	select {
	case <-executor.calls:
	case <-time.After(time.Second):
		require.Fail(t, "a failure waits for the round in progress to end")
	}
	cancel()
	executor.answers <- answer{}
	stopped(t, done)
}

func TestWithNoRoundInProgressTheRunnerWaitsToStop(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	executor, cancel, done := run(t, clock, time.Millisecond)

	<-executor.calls
	executor.answers <- answer{}

	select {
	case <-executor.calls:
		require.Fail(t, "nothing is left to close")
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	stopped(t, done)
}
