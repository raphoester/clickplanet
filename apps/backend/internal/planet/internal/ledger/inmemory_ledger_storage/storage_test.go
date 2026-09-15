package inmemory_ledger_storage_test

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
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

func take(tile uint32, scope, country, previous string, at time.Time) ledger.Taking {
	return ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: previous, At: at}
}

func TestEveryTakeIsKeptInOrder(t *testing.T) {
	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)

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
	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)
	storage.Append(take(1, "1.2.3.4", "fr", "", start.Add(900*time.Millisecond)))

	assert.Equal(t, start, replay(storage)[0].At)
}

func TestTakesSpanChunks(t *testing.T) {
	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)

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
	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)

	storage.Append(take(1, "bot", "ps", "il", start))
	storage.Append(take(2, "player", "il", "", start))
	end := storage.Replay(func(ledger.Taking) {})
	storage.Append(take(3, "bot", "ps", "", start.Add(time.Second)))

	storage.Forget("bot", end)

	assert.Equal(t, []ledger.Taking{
		take(2, "player", "il", "", start),
		take(3, "bot", "ps", "", start.Add(time.Second)),
	}, replay(storage), "a take made after the revert read the ledger is not forgotten")
}

func TestForgetBeforeDropsOnlyOlderTakes(t *testing.T) {
	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)

	storage.Append(take(1, "1.2.3.4", "fr", "", start))
	storage.Append(take(2, "1.2.3.4", "fr", "", start.Add(time.Hour)))

	storage.ForgetBefore(start.Add(time.Minute))

	assert.Equal(t, []ledger.Taking{take(2, "1.2.3.4", "fr", "", start.Add(time.Hour))}, replay(storage))
}

func TestTheCapDropsTheOldestTakes(t *testing.T) {
	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{MaxTakes: 2}, nil)

	storage.Append(take(1, "a", "fr", "", start))
	storage.Append(take(2, "b", "fr", "", start))
	storage.Append(take(3, "c", "fr", "", start))

	takings := replay(storage)
	require.Len(t, takings, 2)
	assert.Equal(t, uint32(2), takings[0].Tile)
}

func TestAReplayRacingAppendsSeesAConsistentPrefix(t *testing.T) {
	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{MaxTakes: 1 << 17}, nil)

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

func statePath(t *testing.T) inmemory_ledger_storage.Config {
	t.Helper()
	return inmemory_ledger_storage.Config{StatePath: filepath.Join(t.TempDir(), "ledger.bin")}
}

func reload(config inmemory_ledger_storage.Config) *inmemory_ledger_storage.Storage {
	storage := inmemory_ledger_storage.New(config, nil)
	storage.LoadState()
	return storage
}

func TestALedgerSurvivesARestartAcrossSeveralSaves(t *testing.T) {
	config := statePath(t)

	before := inmemory_ledger_storage.New(config, nil)
	before.Append(take(1, "1.2.3.4", "fr", "de", start))
	before.Append(take(2, "2001:db8::/64", "fr", "", start.Add(time.Second)))
	require.NoError(t, before.Save())
	size := fileSize(t, config)

	before.Append(take(1, "5.6.7.8", "ps", "fr", start.Add(time.Minute)))
	require.NoError(t, before.Save())
	assert.Greater(t, fileSize(t, config), size, "the second save appends")

	require.NoError(t, before.Save())

	assert.Equal(t, replay(before), replay(reload(config)))
}

func TestAReloadedLedgerKeepsItsPositionsAndForgets(t *testing.T) {
	config := statePath(t)

	before := inmemory_ledger_storage.New(config, nil)
	before.Append(take(1, "bot", "ps", "", start))
	before.Append(take(2, "bot", "ps", "", start.Add(time.Second)))
	require.NoError(t, before.Save())

	before.Forget("bot", before.Replay(func(ledger.Taking) {}))
	require.NoError(t, before.Save())

	after := reload(config)
	assert.Empty(t, replay(after))

	after.Append(take(3, "bot", "ps", "", start.Add(time.Minute)))
	assert.Len(t, replay(after), 1, "positions carry on past the forgotten takes")
}

func TestADroppedTakeStaysDroppedAfterARestart(t *testing.T) {
	config := statePath(t)

	storage := inmemory_ledger_storage.New(config, nil)
	storage.Append(take(1, "1.2.3.4", "fr", "", start))
	storage.Append(take(2, "1.2.3.4", "fr", "", start.Add(time.Hour)))
	require.NoError(t, storage.Save())

	storage.ForgetBefore(start.Add(time.Minute))
	require.NoError(t, storage.Save())

	assert.Equal(t, []ledger.Taking{take(2, "1.2.3.4", "fr", "", start.Add(time.Hour))}, replay(reload(config)))

	storage.ForgetBefore(start.Add(2 * time.Hour))
	require.NoError(t, storage.Save())
	assert.Empty(t, replay(reload(config)))
}

func TestAFileGrownPastWhatItKeepsIsWrittenAgainWhole(t *testing.T) {
	config := statePath(t)

	storage := inmemory_ledger_storage.New(config, nil)
	for i := range 1 << 17 {
		storage.Append(take(uint32(i), "1.2.3.4", "fr", "", start))
	}
	require.NoError(t, storage.Save())
	full := fileSize(t, config)

	storage.ForgetBefore(start.Add(time.Second))
	storage.Append(take(1, "1.2.3.4", "de", "fr", start.Add(time.Hour)))
	require.NoError(t, storage.Save())

	assert.Less(t, fileSize(t, config), full/10)
	assert.Len(t, replay(reload(config)), 1)
}

func TestADamagedTailKeepsTheFramesBeforeIt(t *testing.T) {
	config := statePath(t)

	storage := inmemory_ledger_storage.New(config, nil)
	storage.Append(take(1, "1.2.3.4", "fr", "", start))
	require.NoError(t, storage.Save())
	storage.Append(take(2, "1.2.3.4", "fr", "", start))
	require.NoError(t, storage.Save())

	raw := readFile(t, config)
	writeFile(t, config, raw[:len(raw)-3])

	damaged := reload(config)
	assert.Equal(t, []ledger.Taking{take(1, "1.2.3.4", "fr", "", start)}, replay(damaged))

	damaged.Append(take(3, "1.2.3.4", "fr", "", start))
	require.NoError(t, damaged.Save())
	assert.Len(t, replay(reload(config)), 2, "the next save writes the damage away rather than appending after it")
}

func TestACorruptHeaderStartsEmpty(t *testing.T) {
	config := statePath(t)

	storage := inmemory_ledger_storage.New(config, nil)
	storage.Append(take(1, "1.2.3.4", "fr", "", start))
	require.NoError(t, storage.Save())

	raw := readFile(t, config)
	raw[0] ^= 0xff
	writeFile(t, config, raw)

	assert.Empty(t, replay(reload(config)))
}

func TestAnUnknownVersionStartsEmpty(t *testing.T) {
	config := statePath(t)
	writeFile(t, config, []byte("CPLEDGR\n\x09"))

	assert.Empty(t, replay(reload(config)))
}

func TestAMissingLedgerFileStartsEmpty(t *testing.T) {
	assert.Empty(t, replay(reload(statePath(t))))
}

func TestAVersionOneFileIsReadAsOneTakePerTileAndWrittenOver(t *testing.T) {
	config := statePath(t)
	writeFile(t, config, versionOne([]ledger.Taking{
		take(2, "2001:db8::/64", "ps", "il", start.Add(time.Minute)),
		take(1, "1.2.3.4", "fr", "de", start),
	}))

	storage := reload(config)
	want := []ledger.Taking{
		take(1, "1.2.3.4", "fr", "de", start),
		take(2, "2001:db8::/64", "ps", "il", start.Add(time.Minute)),
	}
	assert.Equal(t, want, replay(storage), "oldest first")

	require.NoError(t, storage.Save())
	assert.Equal(t, byte(2), readFile(t, config)[8])
	assert.Equal(t, want, replay(reload(config)))
}

// versionOne writes the old format: magic, version, CRC32, string table, then 24 bytes per tile.
func versionOne(takings []ledger.Taking) []byte {
	var strs []string
	ids := map[string]uint32{}
	intern := func(value string) uint32 {
		if id, ok := ids[value]; ok {
			return id
		}
		ids[value] = uint32(len(strs))
		strs = append(strs, value)
		return ids[value]
	}

	var records []byte
	for _, taking := range takings {
		records = binary.LittleEndian.AppendUint32(records, taking.Tile)
		records = binary.LittleEndian.AppendUint32(records, intern(taking.Scope))
		records = binary.LittleEndian.AppendUint32(records, intern(taking.Country))
		records = binary.LittleEndian.AppendUint32(records, intern(taking.Previous))
		records = binary.LittleEndian.AppendUint64(records, uint64(taking.At.UnixNano()))
	}

	payload := binary.LittleEndian.AppendUint32(nil, uint32(len(strs)))
	for _, value := range strs {
		payload = binary.LittleEndian.AppendUint16(payload, uint16(len(value)))
		payload = append(payload, value...)
	}
	payload = binary.LittleEndian.AppendUint32(payload, uint32(len(takings)))
	payload = append(payload, records...)

	raw := append([]byte("CPLEDGR\n"), 1)
	raw = binary.LittleEndian.AppendUint32(raw, crc32.ChecksumIEEE(payload))

	return append(raw, payload...)
}

func fileSize(t *testing.T, config inmemory_ledger_storage.Config) int64 {
	t.Helper()
	info, err := os.Stat(config.StatePath)
	require.NoError(t, err)
	return info.Size()
}

func readFile(t *testing.T, config inmemory_ledger_storage.Config) []byte {
	t.Helper()
	raw, err := os.ReadFile(config.StatePath)
	require.NoError(t, err)
	return raw
}

func writeFile(t *testing.T, config inmemory_ledger_storage.Config, raw []byte) {
	t.Helper()
	//nolint:gosec // G703: path is this test's own t.TempDir() file.
	require.NoError(t, os.WriteFile(config.StatePath, raw, 0o600))
}
