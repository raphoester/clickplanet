package inmemory_tile_storage

import (
	"context"
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Kept is a map as postgres keeps it, on a board of its own: the world's while an arena's plays, which no flush writes.
func (s *Storage) Kept(ctx context.Context, kept clicks.MapID) (clicks.DenseBatch, error) {
	s.mu.RLock()
	last := s.board.last()
	s.mu.RUnlock()

	board := newBoard(last)
	var internErr error
	err := s.persistence.Load(ctx, kept, func(tile Tile) {
		if tile.ID > last || internErr != nil {
			return
		}
		owner, err := board.intern(tile.Owner)
		if err != nil {
			internErr = err
			return
		}
		board.restore(tile.ID, tileState{owner: owner, shields: uint8(min(max(tile.Shields, 0), math.MaxUint8))}) //nolint:gosec // clamped to a byte.
	})
	if err != nil {
		return clicks.DenseBatch{}, fmt.Errorf("failed to read the %s map kept: %w", kept, err)
	}
	if internErr != nil {
		return clicks.DenseBatch{}, fmt.Errorf("failed to intern a country of the %s map kept: %w", kept, internErr)
	}
	return board.batch(1, last), nil
}
