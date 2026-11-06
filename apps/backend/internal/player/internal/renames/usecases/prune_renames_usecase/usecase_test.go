package prune_renames_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/renames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/renames/inmemory_rename_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/renames/usecases/prune_renames_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)

func renamed(t *testing.T, store *inmemory_rename_store.Store, at time.Time) {
	t.Helper()

	before := players.ProfileSnapshot{Account: players.AccountID{15: 1}, Name: "Hitler88"}.Restore()
	after := players.ProfileSnapshot{Account: players.AccountID{15: 1}, Name: "Player42"}.Restore()
	require.NoError(t, store.Record(t.Context(), renames.NewRename(before, after, at, "", renames.Operator{})))
}

func TestARenameIsPrunedPastTheRetention(t *testing.T) {
	store := inmemory_rename_store.New()
	renamed(t, store, now.Add(-91*24*time.Hour))
	renamed(t, store, now.Add(-89*24*time.Hour))

	deleted, err := prune_renames_usecase.New(90*24*time.Hour, cptime.NewFixedClock(now), store).Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, int64(1), deleted)
	require.Len(t, store.Kept(), 1)
	assert.Equal(t, now.Add(-89*24*time.Hour), store.Kept()[0].At)
}

func TestAFailedPruneSaysSo(t *testing.T) {
	store := inmemory_rename_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := prune_renames_usecase.New(time.Hour, cptime.NewFixedClock(now), store).Execute(t.Context())

	require.Error(t, err)
}

type countingExecutor struct {
	calls chan struct{}
}

func (c countingExecutor) Execute(context.Context) (int64, error) {
	c.calls <- struct{}{}
	return 0, nil
}

func TestTheRunnerPrunesAtStartThenEveryIntervalUntilStopped(t *testing.T) {
	executor := countingExecutor{calls: make(chan struct{}, 10)}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		prune_renames_usecase.NewRunner(10*time.Millisecond, executor).Run(ctx)
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
