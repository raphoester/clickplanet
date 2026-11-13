package inmemory_tile_storage

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (s *Storage) Restore(_ context.Context, restorations []clicks.Restoration) (int, error) {
	updates := make([]clicks.TileUpdate, 0, len(restorations))

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, restoration := range restorations {
		if restoration.Tile > s.board.last() || restoration.From == restoration.To {
			continue
		}

		if s.board.ownerOf(restoration.Tile) != restoration.From {
			continue
		}

		toID, err := s.board.intern(restoration.To)
		if err != nil {
			s.settleLocked(updates)
			s.publishUpdatesLocked(updates)
			return len(updates), err
		}

		s.moveLocked(restoration.Tile, toID)

		updates = append(updates, clicks.TileUpdate{
			Tile:     restoration.Tile,
			Value:    restoration.To,
			Previous: restoration.From,
		})
	}

	s.settleLocked(updates)
	s.publishUpdatesLocked(updates)

	return len(updates), nil
}

func (s *Storage) publishUpdatesLocked(updates []clicks.TileUpdate) {
	for i := range updates {
		s.publishLocked(clicks.Change{Update: &updates[i]})
	}
}
