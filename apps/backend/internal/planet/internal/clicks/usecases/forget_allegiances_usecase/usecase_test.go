package forget_allegiances_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_allegiance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/forget_allegiances_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

var (
	gone = clicks.AccountAllegianceKey("gone")
	kept = clicks.AccountAllegianceKey("kept")
)

func TestATallyWithNoTakeInThreeDaysIsForgotten(t *testing.T) {
	store := inmemory_allegiance_store.New()
	require.NoError(t, store.SaveAllegiances(t.Context(), map[clicks.AllegianceKey]clicks.Allegiance{
		gone: clicks.Allegiance{}.With("fr", now.Add(-73*time.Hour)),
		kept: clicks.Allegiance{}.With("fr", now.Add(-71*time.Hour)),
	}))

	deleted, err := forget_allegiances_usecase.New(cptime.NewFixedClock(now), store).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
	tallies, err := store.Allegiances(t.Context(), gone, kept)
	require.NoError(t, err)
	assert.Len(t, tallies, 1)
	assert.Contains(t, tallies, kept)
}

func TestAFailedForgetSaysSo(t *testing.T) {
	store := inmemory_allegiance_store.New()
	store.FailWith(errors.New("down"))

	_, err := forget_allegiances_usecase.New(cptime.NewFixedClock(now), store).Execute(t.Context())

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
