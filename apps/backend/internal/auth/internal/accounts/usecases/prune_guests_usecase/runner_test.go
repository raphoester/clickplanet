package prune_guests_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/prune_guests_usecase"
)

type countingExecutor struct {
	calls chan struct{}
}

func (c countingExecutor) Execute(context.Context) (int, error) {
	c.calls <- struct{}{}
	return 0, nil
}

func TestTheRunnerPrunesAtStartThenEveryIntervalUntilStopped(t *testing.T) {
	executor := countingExecutor{calls: make(chan struct{}, 10)}
	runner := prune_guests_usecase.NewRunner(prune_guests_usecase.Config{Interval: 10 * time.Millisecond}, executor)
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
