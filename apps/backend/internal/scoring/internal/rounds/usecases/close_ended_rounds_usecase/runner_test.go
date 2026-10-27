package close_ended_rounds_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_ended_rounds_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type stubExecutor struct {
	report close_ended_rounds_usecase.Report
	err    error
	calls  chan time.Time
}

func (s stubExecutor) Execute(context.Context) (close_ended_rounds_usecase.Report, error) {
	s.calls <- time.Now()
	return s.report, s.err
}

func waitBetweenCalls(t *testing.T, report close_ended_rounds_usecase.Report, err error) time.Duration {
	t.Helper()
	executor := stubExecutor{report: report, err: err, calls: make(chan time.Time, 10)}
	go close_ended_rounds_usecase.NewRunner(executor, cptime.NewFixedClock(utc(16, 12, 0)), 20*time.Millisecond).Run(t.Context())

	first := <-executor.calls
	select {
	case again := <-executor.calls:
		return again.Sub(first)
	case <-time.After(time.Second):
		return time.Second
	}
}

func TestTheRunnerClosesAtStartThenWhenTheRoundInProgressEnds(t *testing.T) {
	wait := waitBetweenCalls(t, close_ended_rounds_usecase.Report{Next: utc(16, 12, 0).Add(100 * time.Millisecond)}, nil)

	assert.GreaterOrEqual(t, wait, 100*time.Millisecond)
	assert.Less(t, wait, time.Second)
}

func TestAFailureIsTriedAgainSoon(t *testing.T) {
	wait := waitBetweenCalls(t, close_ended_rounds_usecase.Report{Next: utc(17, 21, 0)}, errors.New("planet is down"))

	assert.Less(t, wait, 100*time.Millisecond)
}

func TestAWakeRunsAgainAtOnce(t *testing.T) {
	executor := stubExecutor{report: close_ended_rounds_usecase.Report{Next: utc(17, 21, 0)}, calls: make(chan time.Time, 10)}
	runner := close_ended_rounds_usecase.NewRunner(executor, cptime.NewFixedClock(utc(16, 12, 0)), 20*time.Millisecond)
	go runner.Run(t.Context())
	<-executor.calls

	runner.Wake()

	select {
	case <-executor.calls:
	case <-time.After(time.Second):
		t.Fatal("a wake did not run the executor again")
	}
}

func TestWithNothingLeftToEndOnlyAWakeRunsAgain(t *testing.T) {
	wait := waitBetweenCalls(t, close_ended_rounds_usecase.Report{}, nil)

	assert.Equal(t, time.Second, wait, "no call came")
}
