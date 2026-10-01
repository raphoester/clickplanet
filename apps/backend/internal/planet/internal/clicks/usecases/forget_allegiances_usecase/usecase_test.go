package forget_allegiances_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/forget_allegiances_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

type fakeAllegiances struct {
	cutoffs []time.Time
	err     error
}

func (f *fakeAllegiances) DeleteAllegiancesBefore(_ context.Context, cutoff time.Time) (int64, error) {
	f.cutoffs = append(f.cutoffs, cutoff)
	return 2, f.err
}

func TestATallyWithNoTakeInThreeDaysIsForgotten(t *testing.T) {
	store := &fakeAllegiances{}

	deleted, err := forget_allegiances_usecase.New(cptime.NewFixedClock(now), store).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	assert.Equal(t, []time.Time{now.Add(-72 * time.Hour)}, store.cutoffs)
}

func TestAFailedForgetSaysSo(t *testing.T) {
	_, err := forget_allegiances_usecase.New(cptime.NewFixedClock(now), &fakeAllegiances{err: errors.New("down")}).
		Execute(t.Context())

	require.Error(t, err)
}

type countingExecutor struct{ calls chan struct{} }

func (c countingExecutor) Execute(context.Context) (int64, error) {
	c.calls <- struct{}{}
	return 0, nil
}

func TestTheRunnerForgetsAtStartThenEveryIntervalUntilStopped(t *testing.T) {
	executor := countingExecutor{calls: make(chan struct{}, 10)}
	runner := forget_allegiances_usecase.NewRunner(10*time.Millisecond, executor)
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
