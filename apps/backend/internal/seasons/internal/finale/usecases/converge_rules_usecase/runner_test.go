package converge_rules_usecase_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale/usecases/converge_rules_usecase"
)

type countingExecutor struct {
	runs atomic.Int32
	err  error
}

func (c *countingExecutor) Execute(context.Context) error {
	c.runs.Add(1)
	return c.err
}

func TestWhatFollowsTheRulesRunsOnlyOnceTheyHold(t *testing.T) {
	failing := &countingExecutor{err: errors.New("planet is not listening yet")}
	then := &countingExecutor{}
	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan struct{})
	go func() {
		converge_rules_usecase.NewRunner(time.Millisecond, failing, then).Run(ctx)
		close(done)
	}()

	assert.Eventually(t, func() bool { return failing.runs.Load() >= 3 }, time.Second, time.Millisecond)
	cancel()
	<-done

	assert.Zero(t, then.runs.Load())
}

func TestTheRunnerKeepsConverging(t *testing.T) {
	rules := &countingExecutor{}
	then := &countingExecutor{}
	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan struct{})
	go func() {
		converge_rules_usecase.NewRunner(time.Millisecond, rules, then).Run(ctx)
		close(done)
	}()

	assert.Eventually(t, func() bool { return then.runs.Load() >= 3 }, time.Second, time.Millisecond)
	cancel()
	<-done
}
