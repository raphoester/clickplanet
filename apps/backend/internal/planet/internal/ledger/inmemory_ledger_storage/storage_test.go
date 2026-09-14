package inmemory_ledger_storage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
)

var start = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

func TestAPutReplacesTheTilesLastTake(t *testing.T) {
	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)

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
}

func TestForgetLeavesATileTakenAgainSince(t *testing.T) {
	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)

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
	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)

	storage.Put(ledger.Taking{Tile: 1, Scope: "1.2.3.4", At: start})
	storage.Put(ledger.Taking{Tile: 2, Scope: "1.2.3.4", At: start.Add(time.Hour)})

	storage.ForgetBefore(start.Add(time.Minute))

	remaining := storage.TakenBy("1.2.3.4")
	require.Len(t, remaining, 1)
	assert.Equal(t, uint32(2), remaining[0].Tile)
}

func TestALedgerSurvivesARestart(t *testing.T) {
	config := inmemory_ledger_storage.Config{StatePath: filepath.Join(t.TempDir(), "ledger.bin")}

	before := inmemory_ledger_storage.New(config, nil)
	before.Put(ledger.Taking{Tile: 1, Scope: "1.2.3.4", Country: "fr", Previous: "de", At: start})
	before.Put(ledger.Taking{Tile: 2, Scope: "2001:db8::/64", Country: "fr", At: start.Add(time.Second)})
	before.Put(ledger.Taking{Tile: 3, Scope: "1.2.3.4", Country: "ps", Previous: "fr", At: start.Add(time.Minute)})
	require.NoError(t, before.Save())

	after := inmemory_ledger_storage.New(config, nil)
	after.LoadState()

	for _, tile := range []uint32{1, 2, 3} {
		want, _ := before.Last(tile)
		got, found := after.Last(tile)
		require.True(t, found, "tile %d", tile)
		assert.Equal(t, want.Scope, got.Scope)
		assert.Equal(t, want.Country, got.Country)
		assert.Equal(t, want.Previous, got.Previous)
		assert.True(t, want.At.Equal(got.At), "tile %d: %s against %s", tile, want.At, got.At)
	}

	assert.Len(t, after.TakenBy("1.2.3.4"), 2)
	assert.Len(t, after.PaintedWith("fr"), 2)

	after.ForgetBefore(start.Add(30 * time.Second))
	assert.Len(t, after.TakenBy("1.2.3.4"), 1)
}

func TestAnEmptiedLedgerIsSavedEmpty(t *testing.T) {
	config := inmemory_ledger_storage.Config{StatePath: filepath.Join(t.TempDir(), "ledger.bin")}

	storage := inmemory_ledger_storage.New(config, nil)
	storage.Put(ledger.Taking{Tile: 1, Scope: "1.2.3.4", Country: "fr", At: start})
	require.NoError(t, storage.Save())

	storage.ForgetBefore(start.Add(time.Hour))
	require.NoError(t, storage.Save())

	reloaded := inmemory_ledger_storage.New(config, nil)
	reloaded.LoadState()
	_, found := reloaded.Last(1)
	assert.False(t, found)
}

func TestACorruptLedgerFileStartsEmpty(t *testing.T) {
	config := inmemory_ledger_storage.Config{StatePath: filepath.Join(t.TempDir(), "ledger.bin")}

	storage := inmemory_ledger_storage.New(config, nil)
	storage.Put(ledger.Taking{Tile: 1, Scope: "1.2.3.4", Country: "fr", At: start})
	require.NoError(t, storage.Save())

	raw, err := os.ReadFile(config.StatePath)
	require.NoError(t, err)
	raw[len(raw)-1] ^= 0xff
	//nolint:gosec // G703: path is this test's own t.TempDir() file.
	require.NoError(t, os.WriteFile(config.StatePath, raw, 0o600))

	reloaded := inmemory_ledger_storage.New(config, nil)
	reloaded.LoadState()
	_, found := reloaded.Last(1)
	assert.False(t, found)
}

func TestAMissingLedgerFileStartsEmpty(t *testing.T) {
	config := inmemory_ledger_storage.Config{StatePath: filepath.Join(t.TempDir(), "ledger.bin")}

	storage := inmemory_ledger_storage.New(config, nil)
	storage.LoadState()

	assert.Empty(t, storage.TakenBy("1.2.3.4"))
}
