package memory_tile_storage

import (
	"encoding/binary"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
)

// StateBatchDense returns tiles [start, end] without repeating a tile id per
// entry. The interned ids are copied out as stored and the table travels with
// them, so nothing is translated here and the reader needs no country list.
func (s *Storage) StateBatchDense(start uint32, end uint32) (domain.DenseBatch, error) {
	s.tilesMu.RLock()
	defer s.tilesMu.RUnlock()

	if end > s.maxIndex {
		end = s.maxIndex
	}
	if start > end {
		return domain.DenseBatch{}, fmt.Errorf("invalid tile range %d..%d", start, end)
	}

	tiles := make([]byte, 0, (uint64(end-start)+1)*2)
	for _, code := range s.tiles[start : uint64(end)+1] {
		tiles = binary.LittleEndian.AppendUint16(tiles, code)
	}

	codes := make([]string, len(s.codes))
	copy(codes, s.codes)

	return domain.DenseBatch{Start: start, Codes: codes, Tiles: tiles}, nil
}
