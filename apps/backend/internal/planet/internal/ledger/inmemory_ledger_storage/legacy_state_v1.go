package inmemory_ledger_storage

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

// Version 1: magic, version, a CRC32 of the payload, a string table, then 24 bytes per tile.
const (
	headerSizeV1 = headerSize + 4
	recordSizeV1 = 4 + 4 + 4 + 4 + 8
)

func decodeV1(raw []byte) ([]ledger.Taking, error) {
	if len(raw) < headerSizeV1 {
		return nil, fmt.Errorf("%w: file is %d bytes, shorter than the header", errCorruptState, len(raw))
	}

	payload := raw[headerSizeV1:]
	want := binary.LittleEndian.Uint32(raw[headerSize:])
	if got := crc32.ChecksumIEEE(payload); got != want {
		return nil, fmt.Errorf("%w: checksum mismatch (want %08x, got %08x)", errCorruptState, want, got)
	}

	r := reader{buf: payload}
	count := int(r.uint32())
	strs := make([]string, 0, min(count, len(payload)/2))
	for i := 0; i < count && r.err == nil; i++ {
		strs = append(strs, r.string())
	}
	records := int(r.uint32())
	if r.err != nil {
		return nil, r.err
	}
	if len(r.buf) != records*recordSizeV1 {
		return nil, fmt.Errorf("%w: expected %d bytes of records, got %d", errCorruptState, records*recordSizeV1, len(r.buf))
	}

	takings := make([]ledger.Taking, 0, records)
	for range records {
		tile, scope, country, previous := r.uint32(), int(r.uint32()), int(r.uint32()), int(r.uint32())
		at := int64(r.uint64()) //nolint:gosec // written from UnixNano.
		if scope >= len(strs) || country >= len(strs) || previous >= len(strs) {
			return nil, fmt.Errorf("%w: string out of range", errCorruptState)
		}

		takings = append(takings, ledger.Taking{
			Tile: tile, Scope: strs[scope], Country: strs[country], Previous: strs[previous], At: time.Unix(0, at).UTC(),
		})
	}

	return takings, nil
}
