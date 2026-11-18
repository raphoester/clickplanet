package inmemory_tile_storage

import (
	"context"
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Kept is a map as postgres keeps it, on a board of its own: the world's while an arena's plays, which no flush writes,
// on the board of its era.
func (s *Storage) Kept(ctx context.Context, kept clicks.Stage) (clicks.DenseBatch, error) {
	ground, err := s.grounds.GroundOf(kept.Board)
	if err != nil {
		return clicks.DenseBatch{}, fmt.Errorf("failed to read the ground of the %s map kept: %w", kept.Map, err)
	}
	last := ground.Tiles()

	board := newBoard(last)
	var internErr error
	err = s.persistence.Load(ctx, kept.Map, func(tile Tile) {
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
		return clicks.DenseBatch{}, fmt.Errorf("failed to read the %s map kept: %w", kept.Map, err)
	}
	if internErr != nil {
		return clicks.DenseBatch{}, fmt.Errorf("failed to intern a country of the %s map kept: %w", kept.Map, internErr)
	}
	batch := board.batch(1, last)
	batch.Board = kept.Board
	return batch, nil
}
