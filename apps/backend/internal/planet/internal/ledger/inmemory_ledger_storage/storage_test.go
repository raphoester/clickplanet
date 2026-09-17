package inmemory_ledger_storage_test

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
)

var start = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

func replay(storage *inmemory_ledger_storage.Storage) []ledger.Taking {
	var takings []ledger.Taking
	storage.Replay(func(taking ledger.Taking) { takings = append(takings, taking) })
	return takings
}

func newStorage(config inmemory_ledger_storage.Config, persistence inmemory_ledger_storage.Persistence) *inmemory_ledger_storage.Storage {
	return inmemory_ledger_storage.New(config, persistence, slog.New(slog.DiscardHandler))
}

func take(tile uint32, scope, country, previous string, at time.Time) ledger.Taking {
	return ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: previous, At: at}
}

func TestEveryTakeIsKeptInOrder(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	want := []ledger.Taking{
		take(7, "1.2.3.4", "fr", "de", start),
		take(7, "5.6.7.8", "de", "fr", start.Add(time.Second)),
		take(8, "1.2.3.4", "fr", "", start.Add(2*time.Second)),
	}
	for _, taking := range want {
		storage.Append(taking)
	}

	assert.Equal(t, want, replay(storage))
	assert.Equal(t, ledger.Position(3), storage.Replay(func(ledger.Taking) {}))
}

func TestATimeIsKeptToTheSecond(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())
	storage.Append(take(1, "1.2.3.4", "fr", "", start.Add(900*time.Millisecond)))

	assert.Equal(t, start, replay(storage)[0].At)
}

func TestTakesSpanChunks(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	const n = 1<<16 + 10
	for i := range n {
		storage.Append(take(uint32(i), "1.2.3.4", "fr", "", start.Add(time.Duration(i)*time.Second)))
	}

	takings := replay(storage)
	require.Len(t, takings, n)
	assert.Equal(t, uint32(n-1), takings[n-1].Tile)

	storage.ForgetBefore(start.Add((1<<16 + 5) * time.Second))
	takings = replay(storage)
	require.Len(t, takings, 5)
	assert.Equal(t, uint32(1<<16+5), takings[0].Tile)
}

func TestForgetHidesTheScopesTakesBeforeThePositionOnly(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	storage.Append(take(1, "bot", "ps", "il", start))
	storage.Append(take(2, "player", "il", "", start))
	end := storage.Replay(func(ledger.Taking) {})
	storage.Append(take(3, "bot", "ps", "", start.Add(time.Second)))

	storage.Forget(ledger.Caller{Scope: "bot"}, end)

	assert.Equal(t, []ledger.Taking{
		take(2, "player", "il", "", start),
		take(3, "bot", "ps", "", start.Add(time.Second)),
	}, replay(storage), "a take made after the revert read the ledger is not forgotten")
}

func TestForgetBeforeDropsOnlyOlderTakes(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	storage.Append(take(1, "1.2.3.4", "fr", "", start))
	storage.Append(take(2, "1.2.3.4", "fr", "", start.Add(time.Hour)))

	storage.ForgetBefore(start.Add(time.Minute))

	assert.Equal(t, []ledger.Taking{take(2, "1.2.3.4", "fr", "", start.Add(time.Hour))}, replay(storage))
}

func TestTheCapDropsTheOldestTakes(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{MaxTakes: 2}, inmemory_ledger_storage.NewMemoryPersistence())

	storage.Append(take(1, "a", "fr", "", start))
	storage.Append(take(2, "b", "fr", "", start))
	storage.Append(take(3, "c", "fr", "", start))

	takings := replay(storage)
	require.Len(t, takings, 2)
	assert.Equal(t, uint32(2), takings[0].Tile)
}

func TestAReplayRacingAppendsSeesAConsistentPrefix(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{MaxTakes: 1 << 17}, inmemory_ledger_storage.NewMemoryPersistence())

	var wg sync.WaitGroup
	wg.Go(func() {
		for i := range 1 << 18 {
			storage.Append(take(uint32(i), "1.2.3.4", "fr", "", start))
		}
	})

	for range 20 {
		last := int64(-1)
		storage.Replay(func(taking ledger.Taking) {
			assert.Greater(t, int64(taking.Tile), last)
			last = int64(taking.Tile)
		})
		storage.ForgetBefore(start)
	}

	wg.Wait()
}

func TestForgetOnAnAccountHidesItsTakesFromEveryScope(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	guest := take(1, "campus", "ps", "il", start)
	guest.Account = "a-guest"
	elsewhere := take(2, "home", "ps", "il", start)
	elsewhere.Account = "a-guest"
	classmate := take(3, "campus", "ps", "il", start)
	classmate.Account = "a-classmate"
	noAccount := take(4, "campus", "ps", "il", start)

	for _, taking := range []ledger.Taking{guest, elsewhere, classmate, noAccount} {
		storage.Append(taking)
	}
	storage.Forget(ledger.Caller{Account: "a-guest"}, storage.Replay(func(ledger.Taking) {}))

	assert.Equal(t, []ledger.Taking{classmate, noAccount}, replay(storage))
}
