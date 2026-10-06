package inmemory_ledger_storage_test

import (
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
)

var start = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

func replay(storage *inmemory_ledger_storage.Storage) []ledger.Taking {
	var takings []ledger.Taking
	storage.Replay(func(event ledger.Event) {
		event.Replay(func(taking ledger.Taking) { takings = append(takings, taking) })
	})
	return takings
}

func newStorage(config inmemory_ledger_storage.Config, persistence inmemory_ledger_storage.Persistence) *inmemory_ledger_storage.Storage {
	return inmemory_ledger_storage.New(config, persistence, slog.New(slog.DiscardHandler))
}

func take(tile uint32, scope, country, previous string, at time.Time) ledger.Taking {
	return ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: previous, At: at}
}

func bombing(scope string, at time.Time, cleared []uint32, owners ...string) ledger.Bombing {
	return ledger.Bombing{Scope: scope, At: at, Blast: clicks.Blast{CountryID: "de", Cleared: cleared, Owners: owners}}
}

func bombed(tile uint32, scope, previous string, at time.Time) ledger.Taking {
	return ledger.Taking{Tile: tile, Scope: scope, Previous: previous, At: at}
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
	assert.Equal(t, ledger.Position(3), storage.Replay(func(ledger.Event) {}))
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
	end := storage.Replay(func(ledger.Event) {})
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
		storage.Replay(func(event ledger.Event) {
			event.Replay(func(taking ledger.Taking) {
				assert.Greater(t, int64(taking.Tile), last)
				last = int64(taking.Tile)
			})
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
	storage.Forget(ledger.Caller{Account: "a-guest"}, storage.Replay(func(ledger.Event) {}))

	assert.Equal(t, []ledger.Taking{classmate, noAccount}, replay(storage))
}

func TestABombingReplaysAsOneClearPerTileInItsPlace(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	storage.Append(take(1, "a", "fr", "", start))
	storage.Append(bombing("bomber", start.Add(time.Second), []uint32{1, 2}, "fr", "il"))
	storage.Append(take(1, "a", "fr", "", start.Add(2*time.Second)))

	assert.Equal(t, []ledger.Taking{
		take(1, "a", "fr", "", start),
		bombed(1, "bomber", "fr", start.Add(time.Second)),
		bombed(2, "bomber", "il", start.Add(time.Second)),
		take(1, "a", "fr", "", start.Add(2*time.Second)),
	}, replay(storage))
	assert.Equal(t, ledger.Position(3), storage.Replay(func(ledger.Event) {}), "a bombing is one event")
}

func TestABombingKeepsWhatItHitWhateverTheCallerDoesWithItsBlast(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	hit := bombing("bomber", start, []uint32{1}, "fr")
	storage.Append(hit)
	hit.Blast.Cleared[0], hit.Blast.Owners[0] = 9, "jp"

	assert.Equal(t, []ledger.Taking{bombed(1, "bomber", "fr", start)}, replay(storage))
}

func TestForgetHidesTheCallersBombings(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	storage.Append(bombing("bot", start, []uint32{1}, "fr"))
	storage.Append(take(2, "player", "il", "", start))
	storage.Forget(ledger.Caller{Scope: "bot"}, storage.Replay(func(ledger.Event) {}))

	assert.Equal(t, []ledger.Taking{take(2, "player", "il", "", start)}, replay(storage))
}

func TestTheRetentionDropsABombingWithItsTime(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	storage.Append(bombing("old", start, []uint32{1}, "fr"))
	storage.Append(take(2, "a", "fr", "", start.Add(time.Hour)))
	storage.Append(bombing("new", start.Add(2*time.Hour), []uint32{2}, "fr"))

	storage.ForgetBefore(start.Add(time.Minute))

	assert.Equal(t, []ledger.Taking{
		take(2, "a", "fr", "", start.Add(time.Hour)),
		bombed(2, "new", "fr", start.Add(2*time.Hour)),
	}, replay(storage))
}

func TestBombingsSpanChunks(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	const n = 1<<16 + 2
	for i := range uint32(n) {
		if i == 1<<16-1 || i == 1<<16 {
			storage.Append(bombing("bomber", start, []uint32{i}, "fr"))
			continue
		}
		storage.Append(take(i, "1.2.3.4", "fr", "", start))
	}

	takings := replay(storage)
	require.Len(t, takings, n)
	assert.Equal(t, bombed(1<<16-1, "bomber", "fr", start), takings[1<<16-1], "the last event of a chunk")
	assert.Equal(t, bombed(1<<16, "bomber", "fr", start), takings[1<<16], "the first event of the next")
	assert.Equal(t, "fr", takings[n-1].Country)
}

type unwritable struct{}

func (unwritable) Replay(func(ledger.Taking)) {}

func (unwritable) Entry() (ledger.Entry, error) { return ledger.Entry{}, errors.New("no words for it") }

func TestAnEventTheLedgerCannotWriteDownIsNotKept(t *testing.T) {
	storage := newStorage(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence())

	storage.Append(unwritable{})
	storage.Append(take(1, "a", "fr", "", start))

	assert.Equal(t, []ledger.Taking{take(1, "a", "fr", "", start)}, replay(storage))
	assert.Equal(t, ledger.Position(1), storage.Replay(func(ledger.Event) {}))
}
