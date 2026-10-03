package prune_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/usecases/prune_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

type fakePruner struct {
	expired, excess int64
	failExpired     error
	calls           []string
	cutoff          time.Time
	kept            int
}

func (p *fakePruner) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	p.calls = append(p.calls, "before")
	p.cutoff = cutoff
	return p.expired, p.failExpired
}

func (p *fakePruner) DeleteOldestBeyond(_ context.Context, kept int) (int64, error) {
	p.calls = append(p.calls, "beyond")
	p.kept = kept
	return p.excess, nil
}

func TestAPruneDeletesByAgeThenByTheCap(t *testing.T) {
	pruner := &fakePruner{expired: 7, excess: 2}

	pruned, err := prune_usecase.New(72*time.Hour, 1000, cptime.NewFixedClock(now), pruner).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, prune_usecase.Pruned{Expired: 7, Excess: 2}, pruned)
	assert.Equal(t, []string{"before", "beyond"}, pruner.calls, "the cap only takes what the retention kept")
	assert.Equal(t, now.Add(-72*time.Hour), pruner.cutoff)
	assert.Equal(t, 1000, pruner.kept)
}

func TestAFailedPruneSaysSoAndWhatItDid(t *testing.T) {
	cause := errors.New("postgres is down")
	pruner := &fakePruner{expired: 3, failExpired: cause}

	pruned, err := prune_usecase.New(time.Hour, 10, cptime.NewFixedClock(now), pruner).Execute(t.Context())

	require.ErrorIs(t, err, cause)
	assert.Equal(t, prune_usecase.Pruned{Expired: 3}, pruned, "a chunk deleted is deleted")
	assert.Equal(t, []string{"before"}, pruner.calls)
}

type countingExecutor struct {
	calls chan struct{}
}

func (c countingExecutor) Execute(context.Context) (prune_usecase.Pruned, error) {
	c.calls <- struct{}{}
	return prune_usecase.Pruned{}, nil
}

func TestTheRunnerPrunesAtStartThenEveryIntervalUntilStopped(t *testing.T) {
	executor := countingExecutor{calls: make(chan struct{}, 10)}
	runner := prune_usecase.NewRunner(10*time.Millisecond, executor)
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
