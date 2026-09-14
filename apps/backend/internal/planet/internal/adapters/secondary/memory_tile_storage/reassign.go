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
		s.markDirtyLocked(uint32(tile)) //nolint:gosec // tile <= maxIndex, which is a uint32.
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

	s.tilesMu.Unlock()

	s.publishUpdates(updates)

	return next, len(updates), nil
}

// Restore applies each restoration whose tile still holds From, under one lock, and publishes each as an ordinary update.
func (s *Storage) Restore(_ context.Context, restorations []clicks.Restoration) (int, error) {
	updates := make([]clicks.TileUpdate, 0, len(restorations))

	s.tilesMu.Lock()
	for _, restoration := range restorations {
		if restoration.Tile > s.maxIndex || restoration.From == restoration.To {
			continue
		}

		fromID, ok := s.codeIDs[restoration.From]
		if !ok || s.tiles[restoration.Tile] != fromID {
			continue
		}

		toID, err := s.internLocked(restoration.To)
		if err != nil {
			s.tilesMu.Unlock()
			s.publishUpdates(updates)
			return len(updates), err
		}

		if fromID != unownedCode {
			s.counts[fromID]--
		}
		if toID != unownedCode {
			s.counts[toID]++
		}
		s.tiles[restoration.Tile] = toID
		s.markDirtyLocked(restoration.Tile)

		updates = append(updates, clicks.TileUpdate{
			Tile:     restoration.Tile,
			Value:    restoration.To,
			Previous: restoration.From,
		})
	}

	s.tilesMu.Unlock()

	s.publishUpdates(updates)

	return len(updates), nil
}

func (s *Storage) publishUpdates(updates []clicks.TileUpdate) {
	for i := range updates {
		s.publish(clicks.Change{Update: &updates[i]})
	}
}
