package inmemory_allegiance_storage_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/inmemory_allegiance_storage"
)

const (
	alice allegiances.Account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	bob   allegiances.Account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"
)

var at = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

func storageOver(persistence *inmemory_allegiance_storage.MemoryPersistence) *inmemory_allegiance_storage.Storage {
	return inmemory_allegiance_storage.New(inmemory_allegiance_storage.Config{}, persistence, slog.New(slog.DiscardHandler))
}

func pledge(flag string, when time.Time) func(allegiances.Allegiance) (allegiances.Allegiance, error) {
	return func(allegiance allegiances.Allegiance) (allegiances.Allegiance, error) {
		return allegiance.Pledged(flag, when)
	}
}

func TestABootLoadsWhatWasKept(t *testing.T) {
	persistence := inmemory_allegiance_storage.NewMemoryPersistence()
	require.NoError(t, persistence.Save(t.Context(), map[allegiances.Account]allegiances.Snapshot{
		alice: {Flag: "de", ChangedAt: at},
	}))
	storage := storageOver(persistence)

	require.NoError(t, storage.Load(t.Context()))

	assert.Equal(t, allegiances.Snapshot{Flag: "de", ChangedAt: at}, storage.Snapshot(alice))
	require.ErrorIs(t, storage.Change(alice, pledge("fr", at.Add(time.Minute))), allegiances.ErrChangedTooSoon)
}

func TestAFailedLoadRefusesTheBoot(t *testing.T) {
	persistence := inmemory_allegiance_storage.NewMemoryPersistence()
	persistence.FailWith(errors.New("postgres is down"))

	assert.Error(t, storageOver(persistence).Load(t.Context()))
}

func TestAFlushWritesWhatChangedSinceTheLast(t *testing.T) {
	persistence := inmemory_allegiance_storage.NewMemoryPersistence()
	storage := storageOver(persistence)
	require.NoError(t, storage.Change(alice, pledge("fr", at)))
	require.NoError(t, storage.Change(bob, pledge("de", at)))

	require.NoError(t, storage.Flush(t.Context()))

	assert.Equal(t, map[allegiances.Account]allegiances.Snapshot{alice: {Flag: "fr"}, bob: {Flag: "de"}}, persistence.Stored())
}

func TestARefusedChangeKeepsTheAllegiance(t *testing.T) {
	storage := storageOver(inmemory_allegiance_storage.NewMemoryPersistence())
	require.NoError(t, storage.Change(alice, pledge("fr", at)))
	require.NoError(t, storage.Change(alice, pledge("de", at)))

	require.Error(t, storage.Change(alice, pledge("fr", at.Add(time.Minute))))

	assert.Equal(t, allegiances.Snapshot{Flag: "de", ChangedAt: at}, storage.Snapshot(alice))
}

func TestAFailedFlushIsWrittenOnTheNext(t *testing.T) {
	persistence := inmemory_allegiance_storage.NewMemoryPersistence()
	storage := storageOver(persistence)
	require.NoError(t, storage.Change(alice, pledge("fr", at)))

	persistence.FailWith(errors.New("postgres is down"))
	require.Error(t, storage.Flush(t.Context()))

	persistence.Heal()
	require.NoError(t, storage.Flush(t.Context()))

	assert.Equal(t, map[allegiances.Account]allegiances.Snapshot{alice: {Flag: "fr"}}, persistence.Stored())
}
