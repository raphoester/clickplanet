package count_takes_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/count_takes_usecase"
)

type scriptedExecutor struct {
	answers  []error
	caughtUp []bool
	calls    chan time.Time
}

func (s *scriptedExecutor) Execute(context.Context) (count_takes_usecase.Out, error) {
	s.calls <- time.Now()
	if len(s.answers) == 0 {
		return count_takes_usecase.Out{CaughtUp: true}, nil
	}
	err, caughtUp := s.answers[0], s.caughtUp[0]
	s.answers, s.caughtUp = s.answers[1:], s.caughtUp[1:]
	return count_takes_usecase.Out{CaughtUp: caughtUp}, err
}

func run(t *testing.T, executor *scriptedExecutor, interval time.Duration) context.CancelFunc {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		count_takes_usecase.NewRunner(count_takes_usecase.Config{PollInterval: interval}, executor).Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return cancel
}

func TestTheRunnerCountsAgainAtOnceUntilCaughtUpThenWaits(t *testing.T) {
	executor := &scriptedExecutor{answers: []error{nil, nil}, caughtUp: []bool{false, false}, calls: make(chan time.Time, 10)}
	run(t, executor, 200*time.Millisecond)

	first, second, third, fourth := <-executor.calls, <-executor.calls, <-executor.calls, <-executor.calls

	assert.Less(t, second.Sub(first), 100*time.Millisecond, "a batch counted is followed at once")
	assert.Less(t, third.Sub(second), 100*time.Millisecond)
	assert.GreaterOrEqual(t, fourth.Sub(third), 200*time.Millisecond, "caught up, it waits the poll interval")
}

func TestTheRunnerWaitsAfterAFailure(t *testing.T) {
	executor := &scriptedExecutor{answers: []error{errors.New("planet is down")}, caughtUp: []bool{false}, calls: make(chan time.Time, 10)}
	run(t, executor, 200*time.Millisecond)

	first, second := <-executor.calls, <-executor.calls

	assert.GreaterOrEqual(t, second.Sub(first), 200*time.Millisecond)
}

func TestTheRunnerStopsWithItsContext(t *testing.T) {
	executor := &scriptedExecutor{calls: make(chan time.Time, 10)}
	cancel := run(t, executor, time.Hour)
	<-executor.calls

	cancel()
}
