package inmemory_tile_storage

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Wipe empties the map once keep says to, in keep's transaction: every tile blank, with no holder and no shield, and
// no fortify lock. The map starts a new epoch and every stream is cut, so every client reads it again. No flush runs
// meanwhile, so none writes the map before it back.
func (s *Storage) Wipe(ctx context.Context, keep func(ctx context.Context) (bool, error)) (bool, error) {
	s.flushing.Lock()
	defer s.flushing.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	wiped, err := s.persistence.Wipe(ctx, keep)
	if err != nil {
		return false, fmt.Errorf("failed to wipe the stored map: %w", err)
	}
	if !wiped {
		return false, nil
	}

	borders := s.landmasses.borders
	s.board = newBoard(s.board.last())
	s.landmasses = newLandmasses(borders)
	s.recount = newRecount(borders.Landmasses())
	s.version = clicks.MapVersion{
		Epoch: max(uint32(s.clock.Now().Unix()), s.version.Epoch+1), //nolint:gosec // seconds fit a uint32 until 2106.
	}
	s.feed.cutAll()

	return true, nil
}
