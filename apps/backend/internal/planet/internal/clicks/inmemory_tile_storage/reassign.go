package inmemory_tile_storage

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (s *Storage) Reassign(_ context.Context, from, to string, start uint32, limit int) (uint32, int, error) {
	if limit <= 0 {
		return 0, 0, errors.New("reassign limit must be positive")
	}

	s.mu.Lock()

	if from == "" || from == to || s.board.heldBy(from) == 0 {
		s.mu.Unlock()
		return 0, 0, nil
	}

	toID, err := s.board.intern(to)
	if err != nil {
		s.mu.Unlock()
		return 0, 0, err
	}

	var next uint32
	updates := make([]clicks.TileUpdate, 0, limit)

	for tile := start; tile <= s.board.last(); tile++ {
		if s.board.ownerOf(tile) != from {
			continue
		}
		if len(updates) == limit {
			next = tile
			break
		}

		s.moveLocked(tile, toID)
		updates = append(updates, clicks.TileUpdate{Tile: tile, Value: to, Previous: from})
	}

	s.settleLocked(updates)
	s.mu.Unlock()

	s.feed.publishUpdates(updates)

	return next, len(updates), nil
}

func (s *Storage) Restore(_ context.Context, restorations []clicks.Restoration) (int, error) {
	updates := make([]clicks.TileUpdate, 0, len(restorations))

	s.mu.Lock()
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
			s.mu.Unlock()
			s.feed.publishUpdates(updates)
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
	s.mu.Unlock()

	s.feed.publishUpdates(updates)

	return len(updates), nil
}
