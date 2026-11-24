package inmemory_tile_storage

import (
	"context"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (s *Storage) Restore(_ context.Context, restorations []clicks.Restoration) (int, error) {
	updates := make([]clicks.TileUpdate, 0, len(restorations))

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, restoration := range restorations {
		if restoration.Tile > s.board.last() || s.board.ownerOf(restoration.Tile) != restoration.From {
			continue
		}

		before := s.board.tiles[restoration.Tile]
		state := tileState{shields: uint8(min(max(restoration.Shields, 0), math.MaxUint8))} //nolint:gosec // clamped to a byte.
		if restoration.From == restoration.To {
			if int(before.shields) != restoration.Was || restoration.Shields <= restoration.Was {
				continue
			}
			state.owner, state.holder = before.owner, before.holder
		} else {
			to, err := s.board.intern(restoration.To)
			if err != nil {
				s.settleLocked(updates)
				s.publishUpdatesLocked(updates)
				return len(updates), err
			}
			state.owner = to
		}

		s.putLocked(restoration.Tile, state)

		updates = append(updates, clicks.TileUpdate{
			Tile:     restoration.Tile,
			Value:    restoration.To,
			Previous: restoration.From,
			Shields:  int(state.shields),
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
