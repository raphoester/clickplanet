package inmemory_player_storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
)

var (
	start = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	ada   = players.AccountID{15: 1}
	bob   = players.AccountID{15: 2}
)

func loaded(t *testing.T, persistence *inmemory_player_storage.MemoryPersistence) *inmemory_player_storage.Storage {
	t.Helper()

	storage := inmemory_player_storage.New(persistence)
	require.NoError(t, storage.Load(t.Context()))
	return storage
}

func TestWhatWasFlushedIsLoadedByTheNextBoot(t *testing.T) {
	persistence := inmemory_player_storage.NewMemoryPersistence()
	storage := loaded(t, persistence)

	storage.SaveProfile(players.Profile{Account: ada, Name: "Ada", UpdatedAt: start})
	storage.RecordTake(ada, start)
	storage.RecordTake(bob, start)
	require.NoError(t, storage.Flush(t.Context()))

	again := loaded(t, persistence)
	profile, ok := again.Profile(ada)
	require.True(t, ok)
	assert.Equal(t, players.Name("Ada"), profile.Name)
	stats, ok := again.Stats(bob)
	require.True(t, ok)
	assert.Equal(t, players.Stats{}.WithTake(start), withoutAccount(stats))
}

func withoutAccount(stats players.Stats) players.Stats {
	stats.Account = players.AccountID{}
	return stats
}

func TestManyChangesBetweenFlushesAreOneSave(t *testing.T) {
	persistence := inmemory_player_storage.NewMemoryPersistence()
	storage := loaded(t, persistence)

	for i := range 50 {
		storage.RecordTake(ada, start.Add(time.Duration(i)*time.Second))
	}
	require.NoError(t, storage.Flush(t.Context()))
	require.NoError(t, storage.Flush(t.Context()))

	assert.Equal(t, 1, persistence.Saves(), "a flush with nothing changed writes nothing")
	stats, _ := loaded(t, persistence).Stats(ada)
	assert.Equal(t, uint64(50), stats.TilesTaken)
}

func TestAFailedFlushIsWrittenByTheNextOne(t *testing.T) {
	persistence := inmemory_player_storage.NewMemoryPersistence()
	storage := loaded(t, persistence)
	storage.RecordTake(ada, start)

	persistence.FailWith(errors.New("postgres is down"))
	require.Error(t, storage.Flush(t.Context()))

	persistence.Heal()
	require.NoError(t, storage.Flush(t.Context()))

	_, ok := loaded(t, persistence).Stats(ada)
	assert.True(t, ok)
}

func TestADeletedAccountIsDeletedFromTheNextBoot(t *testing.T) {
	persistence := inmemory_player_storage.NewMemoryPersistence()
	storage := loaded(t, persistence)
	storage.SaveProfile(players.Profile{Account: ada, Name: "Ada", UpdatedAt: start})
	storage.RecordTake(ada, start)
	storage.SaveProfile(players.Profile{Account: bob, Name: "Bob", UpdatedAt: start})
	require.NoError(t, storage.Flush(t.Context()))

	storage.DeleteAccount(ada)
	_, named := storage.Profile(ada)
	require.False(t, named)
	require.NoError(t, storage.Flush(t.Context()))

	again := loaded(t, persistence)
	_, named = again.Profile(ada)
	assert.False(t, named)
	_, played := again.Stats(ada)
	assert.False(t, played)
	assert.Equal(t, map[players.AccountID]players.Name{bob: "Bob"}, again.Names([]players.AccountID{ada, bob}))
}

func TestAFailedLoadIsAnError(t *testing.T) {
	persistence := inmemory_player_storage.NewMemoryPersistence()
	persistence.FailWith(errors.New("postgres is down"))

	assert.Error(t, inmemory_player_storage.New(persistence).Load(t.Context()))
}

// countingFlusher sends whether each flush's context was already done.
type countingFlusher struct {
	flushes chan error
}

func (c countingFlusher) Flush(ctx context.Context) error {
	c.flushes <- ctx.Err()
	return nil
}

func TestTheRunnerFlushesEveryIntervalAndOnceMoreWhenItStops(t *testing.T) {
	flusher := countingFlusher{flushes: make(chan error, 10)}
	runner := inmemory_player_storage.NewRunner(inmemory_player_storage.Config{FlushInterval: 10 * time.Millisecond}, flusher)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		runner.Run(ctx)
		close(done)
	}()
	<-flusher.flushes
	cancel()
	<-done

	require.NotEmpty(t, flusher.flushes, "the last flush happens after the runner is told to stop")
	var last error
	for len(flusher.flushes) > 0 {
		last = <-flusher.flushes
	}
	assert.NoError(t, last, "and it is not cancelled with the process")
}
