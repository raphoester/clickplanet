package inmemory_ledger_storage_test

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
)

func legacyConfig(t *testing.T, raw []byte) inmemory_ledger_storage.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ledger.bin")
	//nolint:gosec // G703: path is this test's own t.TempDir() file.
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	return inmemory_ledger_storage.Config{LegacyStatePath: path}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestAVersionTwoFileIsImportedWithItsPositionsAndMarks(t *testing.T) {
	config := legacyConfig(t, versionTwo(
		takesFrame(0, take(1, "old", "fr", "", start), take(2, "bot", "ps", "il", start)),
		marksFrame(1, map[string]ledger.Position{"bot": 3}),
		takesFrame(3, take(3, "bot", "ps", "", start.Add(time.Minute)), take(4, "player", "de", "", start.Add(time.Hour))),
	))
	persistence := inmemory_ledger_storage.NewMemoryPersistence()

	storage := loaded(t, config, persistence)
	assert.Equal(t, []ledger.Taking{
		take(3, "bot", "ps", "", start.Add(time.Minute)),
		take(4, "player", "de", "", start.Add(time.Hour)),
	}, replay(storage), "a take before the head is gone, and one before its scope's mark is forgotten")

	require.NoError(t, storage.Flush(t.Context()))

	assert.Equal(t, []ledger.Position{1, 3, 4}, positions(persistence.Stored()))
	assert.Equal(t, inmemory_ledger_storage.Marks{Head: 1, Forgotten: map[string]ledger.Position{"bot": 3}}, persistence.Marks())
	assert.False(t, exists(config.LegacyStatePath))
	assert.True(t, exists(config.LegacyStatePath+".imported"))
	assert.Equal(t, replay(storage), replay(loaded(t, inmemory_ledger_storage.Config{}, persistence)))
}

func TestTheFileIsRenamedOnlyOnceAFlushCommits(t *testing.T) {
	config := legacyConfig(t, versionTwo(takesFrame(0, take(1, "a", "fr", "", start))))
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	storage := loaded(t, config, persistence)

	persistence.FailWith(errors.New("connection reset"))
	require.Error(t, storage.Flush(t.Context()))
	assert.True(t, exists(config.LegacyStatePath))

	persistence.Heal()
	require.NoError(t, storage.Flush(t.Context()))
	assert.False(t, exists(config.LegacyStatePath))
	assert.Len(t, persistence.Stored(), 1)
}

func TestABootThatNeverFlushedImportsAgain(t *testing.T) {
	config := legacyConfig(t, versionTwo(takesFrame(0, take(1, "a", "fr", "", start))))
	persistence := inmemory_ledger_storage.NewMemoryPersistence()

	loaded(t, config, persistence)
	again := loaded(t, config, persistence)

	assert.Equal(t, []ledger.Taking{take(1, "a", "fr", "", start)}, replay(again))
}

func TestAnEmptyFileIsRenamedAtTheFirstFlush(t *testing.T) {
	config := legacyConfig(t, versionTwo())
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	storage := loaded(t, config, persistence)

	require.NoError(t, storage.Flush(t.Context()))

	assert.False(t, exists(config.LegacyStatePath))
}

func TestAStoredLedgerIgnoresTheFile(t *testing.T) {
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	first := loaded(t, inmemory_ledger_storage.Config{}, persistence)
	first.Append(take(9, "stored", "fr", "", start))
	require.NoError(t, first.Flush(t.Context()))

	config := legacyConfig(t, versionTwo(takesFrame(0, take(1, "file", "de", "", start))))
	storage := loaded(t, config, persistence)
	require.NoError(t, storage.Flush(t.Context()))

	assert.Equal(t, []ledger.Taking{take(9, "stored", "fr", "", start)}, replay(storage))
	assert.True(t, exists(config.LegacyStatePath))
}

func TestADamagedTailImportsTheFramesBeforeIt(t *testing.T) {
	raw := versionTwo(takesFrame(0, take(1, "a", "fr", "", start)), takesFrame(1, take(2, "a", "fr", "", start)))
	config := legacyConfig(t, raw[:len(raw)-3])

	storage := loaded(t, config, inmemory_ledger_storage.NewMemoryPersistence())

	assert.Equal(t, []ledger.Taking{take(1, "a", "fr", "", start)}, replay(storage))
}

func TestAFileItCannotDecodeRefusesTheBoot(t *testing.T) {
	for name, raw := range map[string][]byte{
		"bad magic":       []byte("NOTALEDG\x02"),
		"unknown version": []byte("CPLEDGR\n\x09"),
		"damaged first":   versionTwo(takesFrame(0, take(1, "a", "fr", "", start)))[:20],
	} {
		t.Run(name, func(t *testing.T) {
			storage := newStorage(legacyConfig(t, raw), inmemory_ledger_storage.NewMemoryPersistence())

			require.ErrorContains(t, storage.Load(t.Context()), "move it away to start empty")
		})
	}
}

func TestAMissingFileStartsEmpty(t *testing.T) {
	config := inmemory_ledger_storage.Config{LegacyStatePath: filepath.Join(t.TempDir(), "ledger.bin")}

	assert.Empty(t, replay(loaded(t, config, inmemory_ledger_storage.NewMemoryPersistence())))
}

func TestAVersionOneFileIsImportedAsOneTakePerTile(t *testing.T) {
	config := legacyConfig(t, versionOne([]ledger.Taking{
		take(2, "2001:db8::/64", "ps", "il", start.Add(time.Minute)),
		take(1, "1.2.3.4", "fr", "de", start),
	}))
	persistence := inmemory_ledger_storage.NewMemoryPersistence()

	storage := loaded(t, config, persistence)
	want := []ledger.Taking{
		take(1, "1.2.3.4", "fr", "de", start),
		take(2, "2001:db8::/64", "ps", "il", start.Add(time.Minute)),
	}
	assert.Equal(t, want, replay(storage), "oldest first")

	require.NoError(t, storage.Flush(t.Context()))
	assert.Equal(t, want, replay(loaded(t, inmemory_ledger_storage.Config{}, persistence)))
}

func positions(stored []inmemory_ledger_storage.Stored) []ledger.Position {
	out := make([]ledger.Position, len(stored))
	for i, s := range stored {
		out[i] = s.Position
	}
	return out
}

// versionTwo writes the old format: magic, version, then frames of a kind, a length, a CRC32 and a payload.
func versionTwo(frames ...[]byte) []byte {
	raw := append([]byte("CPLEDGR\n"), 2)
	for _, frame := range frames {
		raw = append(raw, frame...)
	}
	return raw
}

func frame(kind byte, payload []byte) []byte {
	raw := []byte{kind}
	raw = binary.LittleEndian.AppendUint32(raw, uint32(len(payload)))
	raw = binary.LittleEndian.AppendUint32(raw, crc32.ChecksumIEEE(payload))
	return append(raw, payload...)
}

func appendString(buf []byte, value string) []byte {
	buf = binary.LittleEndian.AppendUint16(buf, uint16(len(value)))
	return append(buf, value...)
}

// takesFrame holds takings at first and the positions after it, with its own string table.
func takesFrame(first ledger.Position, takings ...ledger.Taking) []byte {
	var strs []string
	ids := map[string]uint32{}
	intern := func(value string) uint32 {
		if _, ok := ids[value]; !ok {
			ids[value] = uint32(len(strs))
			strs = append(strs, value)
		}
		return ids[value]
	}

	var records []byte
	for _, taking := range takings {
		records = binary.LittleEndian.AppendUint32(records, uint32(taking.At.Unix()))
		records = binary.LittleEndian.AppendUint32(records, taking.Tile)
		records = binary.LittleEndian.AppendUint32(records, intern(taking.Scope))
		records = binary.LittleEndian.AppendUint16(records, uint16(intern(taking.Country)))
		records = binary.LittleEndian.AppendUint16(records, uint16(intern(taking.Previous)))
	}

	payload := binary.LittleEndian.AppendUint64(nil, uint64(first))
	for range 2 {
		payload = binary.LittleEndian.AppendUint32(payload, uint32(len(strs)))
		for _, value := range strs {
			payload = appendString(payload, value)
		}
	}
	payload = binary.LittleEndian.AppendUint32(payload, uint32(len(takings)))

	return frame(1, append(payload, records...))
}

func marksFrame(head ledger.Position, forgotten map[string]ledger.Position) []byte {
	payload := binary.LittleEndian.AppendUint64(nil, uint64(head))
	payload = binary.LittleEndian.AppendUint32(payload, uint32(len(forgotten)))
	for scope, before := range forgotten {
		payload = appendString(payload, scope)
		payload = binary.LittleEndian.AppendUint64(payload, uint64(before))
	}
	return frame(2, payload)
}

// versionOne writes the oldest format: magic, version, CRC32, string table, then 24 bytes per tile.
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
		payload = appendString(payload, value)
	}
	payload = binary.LittleEndian.AppendUint32(payload, uint32(len(takings)))
	payload = append(payload, records...)

	raw := append([]byte("CPLEDGR\n"), 1)
	raw = binary.LittleEndian.AppendUint32(raw, crc32.ChecksumIEEE(payload))

	return append(raw, payload...)
}
