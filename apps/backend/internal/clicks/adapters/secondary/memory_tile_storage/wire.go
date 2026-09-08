package memory_tile_storage

import (
	"encoding/binary"
	"fmt"
)

// Wire layout of a map chunk — the same shape as the snapshot payload, minus
// the checksum, because HTTP already frames and checks the body:
//
//	magic     4 bytes  "CPM1"
//	codeCount 2 bytes  uint16, little endian
//	codes     codeCount x (1 byte length + that many bytes), index 0 is ""
//	startTile 4 bytes  uint32, little endian
//	tileCount 4 bytes  uint32, little endian
//	tiles     tileCount x 2 bytes, little endian index into codes
//
// Tile ids are implicit: the nth tile of the body is startTile+n. That is what
// makes this about five times smaller than a protobuf map<uint32, string>,
// which repeats the id the position already carries.
const mapChunkMagic = "CPM1"

// EncodeStateBatch returns the dense encoding of tiles [start, end].
//
// The interned ids are written as they are stored, and the interning table
// travels with them, so the encoder never translates a tile and the reader
// never needs a shared country list.
func (s *Storage) EncodeStateBatch(start uint32, end uint32) ([]byte, error) {
	s.tilesMu.RLock()
	defer s.tilesMu.RUnlock()

	if end > s.maxIndex {
		end = s.maxIndex
	}
	if start > end {
		return nil, fmt.Errorf("invalid tile range %d..%d", start, end)
	}

	count := uint32(end-start) + 1

	size := len(mapChunkMagic) + 2 + 4 + 4 + int(count)*2
	for _, code := range s.codes {
		size += 1 + len(code)
	}

	buf := make([]byte, 0, size)
	buf = append(buf, mapChunkMagic...)
	buf = binary.LittleEndian.AppendUint16(buf, uint16(len(s.codes)))
	for _, code := range s.codes {
		buf = append(buf, uint8(len(code)))
		buf = append(buf, code...)
	}

	buf = binary.LittleEndian.AppendUint32(buf, start)
	buf = binary.LittleEndian.AppendUint32(buf, count)
	for _, code := range s.tiles[start : uint64(end)+1] {
		buf = binary.LittleEndian.AppendUint16(buf, code)
	}

	return buf, nil
}
