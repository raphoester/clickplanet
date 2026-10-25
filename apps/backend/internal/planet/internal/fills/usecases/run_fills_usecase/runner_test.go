package run_fills_usecase_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/usecases/run_fills_usecase"
)

type executor struct {
	runs    atomic.Int32
	failing atomic.Int32
}

func (e *executor) Execute(context.Context) ([]fills.Fill, error) {
	e.runs.Add(1)
	if e.failing.Add(-1) >= 0 {
		return nil, errors.New("postgres is down")
	}
	return nil, nil
}

func running(t *testing.T, runner *run_fills_usecase.Runner) func() {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		runner.Run(ctx)
		close(done)
	}()
	return func() {
		cancel()
		<-done
	}
}

func TestTheRunnerRunsAtBootThenEachTimeItIsWoken(t *testing.T) {
	work := &executor{}
	runner := run_fills_usecase.NewRunner(work, time.Hour)
	stop := running(t, runner)
	defer stop()

	assert.Eventually(t, func() bool { return work.runs.Load() == 1 }, time.Second, time.Millisecond)

	runner.Wake()
	assert.Eventually(t, func() bool { return work.runs.Load() == 2 }, time.Second, time.Millisecond)
	assert.Never(t, func() bool { return work.runs.Load() > 2 }, 50*time.Millisecond, time.Millisecond)
}

func TestTheRunnerTriesAgainAfterAFailure(t *testing.T) {
	work := &executor{}
	work.failing.Store(2)
	stop := running(t, run_fills_usecase.NewRunner(work, 10*time.Millisecond))
	defer stop()

	assert.Eventually(t, func() bool { return work.runs.Load() == 3 }, time.Second, time.Millisecond)
	assert.Never(t, func() bool { return work.runs.Load() > 3 }, 50*time.Millisecond, time.Millisecond)
}

func TestTheRunnerStopsWithItsContext(t *testing.T) {
	stop := running(t, run_fills_usecase.NewRunner(&executor{}, time.Hour))

	stopped := make(chan struct{})
	go func() {
		stop()
		close(stopped)
	}()

	assert.Eventually(t, func() bool {
		select {
		case <-stopped:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond, "the runner stops once its context is done")
}
