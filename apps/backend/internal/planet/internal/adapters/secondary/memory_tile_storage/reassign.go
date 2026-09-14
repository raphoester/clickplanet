package memory_tile_storage

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (s *Storage) Held(country string) int {
	s.tilesMu.RLock()
	defer s.tilesMu.RUnlock()

	id, ok := s.codeIDs[country]
	if !ok || id == unownedCode {
		return 0
	}

	return int(s.counts[id])
}

// Reassign moves up to limit of from's tiles to to, scanning from start; next is 0 once the map is done.
func (s *Storage) Reassign(_ context.Context, from, to string, start uint32, limit int) (uint32, int, error) {
	if limit <= 0 {
		return 0, 0, errors.New("reassign limit must be positive")
	}

	s.tilesMu.Lock()

	fromID, held := s.codeIDs[from]
	if !held || fromID == unownedCode || from == to {
		s.tilesMu.Unlock()
		return 0, 0, nil
	}

	toID, err := s.internLocked(to)
	if err != nil {
		s.tilesMu.Unlock()
		return 0, 0, err
	}

	var next uint32
	updates := make([]clicks.TileUpdate, 0, limit)

	for tile := int(start); tile <= int(s.maxIndex); tile++ {
		if s.tiles[tile] != fromID {
			continue
		}
		if len(updates) == limit {
			next = uint32(tile) //nolint:gosec // tile <= maxIndex, which is a uint32.
			break
		}

		s.tiles[tile] = toID
		s.counts[fromID]--
		if toID != unownedCode {
			s.counts[toID]++
		}
		updates = append(updates, clicks.TileUpdate{
			Tile:     uint32(tile), //nolint:gosec // tile <= maxIndex, which is a uint32.
			Value:    to,
			Previous: from,
		})
	}

	if len(updates) > 0 {
		s.dirty = true
	}
	s.tilesMu.Unlock()

	for i := range updates {
		s.publish(clicks.Change{Update: &updates[i]})
	}

	return next, len(updates), nil
}
