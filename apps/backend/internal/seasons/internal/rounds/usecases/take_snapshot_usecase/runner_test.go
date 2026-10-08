package take_snapshot_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/take_snapshot_usecase"
)

type countingExecutor struct {
	calls chan struct{}
}

func (c countingExecutor) Execute(context.Context) ([]rounds.Closed, error) {
	c.calls <- struct{}{}
	return nil, nil
}

func TestTheRunnerTakesASnapshotAtStartThenEveryIntervalUntilStopped(t *testing.T) {
	executor := countingExecutor{calls: make(chan struct{}, 10)}
	runner := take_snapshot_usecase.NewRunner(take_snapshot_usecase.Config{Interval: 10 * time.Millisecond}, executor)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		runner.Run(ctx)
		close(done)
	}()

	<-executor.calls
	<-executor.calls
	cancel()
	assert.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
}

func TestTheIntervalIsAMinuteUnlessSet(t *testing.T) {
	assert.Equal(t, time.Minute, take_snapshot_usecase.Config{}.WithDefaults().Interval)
	assert.Equal(t, time.Second, take_snapshot_usecase.Config{Interval: time.Second}.WithDefaults().Interval)
}
