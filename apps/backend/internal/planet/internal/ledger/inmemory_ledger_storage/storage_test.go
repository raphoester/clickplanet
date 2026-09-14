package inmemory_ledger_storage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
)

var start = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

func TestAPutReplacesTheTilesLastTake(t *testing.T) {
	storage := inmemory_ledger_storage.New()

	storage.Put(ledger.Taking{Tile: 7, Scope: "1.2.3.4", Country: "fr", Previous: "de", At: start})
	storage.Put(ledger.Taking{Tile: 7, Scope: "5.6.7.8", Country: "de", Previous: "fr", At: start})

	last, found := storage.Last(7)
	require.True(t, found)
	assert.Equal(t, "5.6.7.8", last.Scope)

	assert.Empty(t, storage.TakenBy("1.2.3.4"))
	assert.Len(t, storage.TakenBy("5.6.7.8"), 1)
	assert.Empty(t, storage.PaintedWith("fr"))
	assert.Len(t, storage.PaintedWith("de"), 1)

	_, found = storage.Last(8)
	assert.False(t, found)
	assert.Len(t, storage.All(), 1)
}

func TestForgetLeavesATileTakenAgainSince(t *testing.T) {
	storage := inmemory_ledger_storage.New()

	storage.Put(ledger.Taking{Tile: 1, Scope: "1.2.3.4", Country: "fr", At: start})
	storage.Put(ledger.Taking{Tile: 2, Scope: "1.2.3.4", Country: "fr", At: start})
	taken := storage.TakenBy("1.2.3.4")

	storage.Put(ledger.Taking{Tile: 2, Scope: "1.2.3.4", Country: "ps", At: start.Add(time.Second)})

	storage.Forget(taken)

	remaining := storage.TakenBy("1.2.3.4")
	require.Len(t, remaining, 1)
	assert.Equal(t, uint32(2), remaining[0].Tile)
}

func TestForgetBeforeDropsOnlyOlderTakes(t *testing.T) {
	storage := inmemory_ledger_storage.New()

	storage.Put(ledger.Taking{Tile: 1, Scope: "1.2.3.4", At: start})
	storage.Put(ledger.Taking{Tile: 2, Scope: "1.2.3.4", At: start.Add(time.Hour)})

	storage.ForgetBefore(start.Add(time.Minute))

	remaining := storage.TakenBy("1.2.3.4")
	require.Len(t, remaining, 1)
	assert.Equal(t, uint32(2), remaining[0].Tile)
}
