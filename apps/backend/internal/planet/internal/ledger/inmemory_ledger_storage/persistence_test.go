package inmemory_ledger_storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
)

func loaded(t *testing.T, config inmemory_ledger_storage.Config, persistence inmemory_ledger_storage.Persistence) *inmemory_ledger_storage.Storage {
	t.Helper()
	storage := newStorage(config, persistence)
	require.NoError(t, storage.Load(t.Context()))
	return storage
}

func TestAnEmptyStoreLoadsAnEmptyLedger(t *testing.T) {
	storage := loaded(t, inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	assert.Empty(t, replay(storage))
	assert.Equal(t, ledger.Position(0), storage.Replay(func(ledger.Taking) {}))
}

func TestALedgerSurvivesARestartWithItsPositionsAndMarks(t *testing.T) {
	ctx := t.Context()
	persistence := inmemory_ledger_storage.NewMemoryPersistence()

	before := loaded(t, inmemory_ledger_storage.Config{}, persistence)
	before.Append(take(1, "bot", "ps", "il", start))
	before.Append(take(2, "player", "fr", "", start.Add(time.Second)))
	require.NoError(t, before.Flush(ctx))
	before.Forget(ledger.Caller{Scope: "bot"}, before.Replay(func(ledger.Taking) {}))
	before.Append(take(3, "bot", "ps", "", start.Add(time.Minute)))
	require.NoError(t, before.Flush(ctx))

	after := loaded(t, inmemory_ledger_storage.Config{}, persistence)
	assert.Equal(t, replay(before), replay(after))
	assert.Equal(t, ledger.Position(3), after.Replay(func(ledger.Taking) {}))

	after.Append(take(4, "bot", "ps", "", start.Add(time.Hour)))
	assert.Len(t, replay(after), 3, "positions carry on past the stored takes")
}

func TestAFailedLoadRefusesTheBoot(t *testing.T) {
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	persistence.FailWith(errors.New("connection refused"))

	storage := newStorage(inmemory_ledger_storage.Config{}, persistence)

	require.ErrorContains(t, storage.Load(t.Context()), "connection refused")
}

func TestFlushWritesOnlyTheTakesSinceTheLastFlush(t *testing.T) {
	ctx := t.Context()
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	storage := loaded(t, inmemory_ledger_storage.Config{}, persistence)

	storage.Append(take(1, "a", "fr", "", start))
	require.NoError(t, storage.Flush(ctx))
	storage.Append(take(2, "b", "de", "", start))
	require.NoError(t, storage.Flush(ctx))
	require.NoError(t, storage.Flush(ctx))

	assert.Equal(t, 2, persistence.Saves(), "a flush with nothing new writes nothing")
	assert.Equal(t, []inmemory_ledger_storage.Stored{
		{Position: 0, Taking: take(1, "a", "fr", "", start)},
		{Position: 1, Taking: take(2, "b", "de", "", start)},
	}, persistence.Stored())
}

func TestAFailedFlushKeepsTheChangesForTheNextOne(t *testing.T) {
	ctx := t.Context()
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	storage := loaded(t, inmemory_ledger_storage.Config{}, persistence)

	storage.Append(take(1, "bot", "ps", "", start))
	storage.Forget(ledger.Caller{Scope: "bot"}, storage.Replay(func(ledger.Taking) {}))
	persistence.FailWith(errors.New("connection reset"))
	require.Error(t, storage.Flush(ctx))

	persistence.Heal()
	storage.Append(take(2, "player", "fr", "", start))
	require.NoError(t, storage.Flush(ctx))

	assert.Len(t, persistence.Stored(), 2)
	assert.Equal(t, map[ledger.Caller]ledger.Position{{Scope: "bot"}: 1}, persistence.Marks().Forgotten)
}

func TestFlushDeletesWhatTheRetentionDropped(t *testing.T) {
	ctx := t.Context()
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	storage := loaded(t, inmemory_ledger_storage.Config{}, persistence)

	storage.Append(take(1, "a", "fr", "", start))
	storage.Append(take(2, "a", "fr", "", start.Add(time.Hour)))
	require.NoError(t, storage.Flush(ctx))

	storage.ForgetBefore(start.Add(time.Minute))
	require.NoError(t, storage.Flush(ctx))

	assert.Equal(t, []inmemory_ledger_storage.Stored{{Position: 1, Taking: take(2, "a", "fr", "", start.Add(time.Hour))}}, persistence.Stored())
	assert.Equal(t, ledger.Position(1), persistence.Marks().Head)
}

func TestATakeDroppedBeforeItWasFlushedIsNeverWritten(t *testing.T) {
	ctx := t.Context()
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	storage := loaded(t, inmemory_ledger_storage.Config{MaxTakes: 1}, persistence)

	storage.Append(take(1, "a", "fr", "", start))
	storage.Append(take(2, "b", "fr", "", start))
	require.NoError(t, storage.Flush(ctx))

	assert.Equal(t, []inmemory_ledger_storage.Stored{{Position: 1, Taking: take(2, "b", "fr", "", start)}}, persistence.Stored())
}

func TestPositionsCarryOnPastALedgerTheRetentionEmptied(t *testing.T) {
	ctx := t.Context()
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	storage := loaded(t, inmemory_ledger_storage.Config{}, persistence)

	storage.Append(take(1, "a", "fr", "", start))
	storage.Append(take(2, "a", "fr", "", start))
	storage.ForgetBefore(start.Add(time.Hour))
	require.NoError(t, storage.Flush(ctx))

	after := loaded(t, inmemory_ledger_storage.Config{}, persistence)
	assert.Equal(t, ledger.Position(2), after.Replay(func(ledger.Taking) {}))
}

func TestRunFlushesOnShutdown(t *testing.T) {
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	storage := loaded(t, inmemory_ledger_storage.Config{FlushInterval: time.Hour}, persistence)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		storage.Run(ctx)
	}()

	storage.Append(take(7, "a", "fr", "", start))
	cancel()

	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond, "Run did not return after the context was cancelled")

	assert.Len(t, persistence.Stored(), 1)
}

func TestRunFlushesOnItsInterval(t *testing.T) {
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	storage := loaded(t, inmemory_ledger_storage.Config{FlushInterval: 10 * time.Millisecond}, persistence)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go storage.Run(ctx)

	storage.Append(take(7, "a", "fr", "", start))

	assert.Eventually(t, func() bool { return len(persistence.Stored()) == 1 }, 5*time.Second, 10*time.Millisecond)
}
