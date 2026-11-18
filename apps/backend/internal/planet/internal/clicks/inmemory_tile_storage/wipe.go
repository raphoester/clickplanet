package inmemory_tile_storage

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Wipe empties the map in play once keep says to, in keep's transaction: every tile blank, with no holder and no
// shield, no fortify lock and no flag out. The map starts a new epoch and every stream is cut, so every client reads it
// again. What changed is saved first, so every shield placed is confirmed. No flush runs meanwhile, so none writes the
// map before it
// back.
func (s *Storage) Wipe(ctx context.Context, keep func(ctx context.Context) (bool, error)) (bool, error) {
	s.flushing.Lock()
	defer s.flushing.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.flushLocked(ctx); err != nil {
		return false, fmt.Errorf("failed to save the map before wiping it: %w", err)
	}
	wiped, err := s.persistence.Wipe(ctx, s.stage.Map, keep)
	if err != nil {
		return false, fmt.Errorf("failed to wipe the stored map: %w", err)
	}
	if !wiped {
		return false, nil
	}

	s.emptyLocked()
	s.restartLocked()

	return true, nil
}

// A blank board, no landmass or province held, no province locked, no flag out, nothing to recount.
func (s *Storage) emptyLocked() {
	s.groundLocked(s.played.Ground())
}

// groundLocked empties the map onto ground, the board of the stage, and plays it: every rule reads it from then on. No
// ground is a map of no tile.
func (s *Storage) groundLocked(ground *clicks.Ground) {
	borders, table := clicks.NoBorders(), clicks.NoProvinces()
	if ground != nil {
		borders, table = ground.Borders(), ground.Provinces()
	}
	s.board = newBoard(borders.Tiles())
	s.landmasses = newLandmasses(borders)
	s.provinces = newProvinces(table)
	s.out = newOutFlags()
	s.viruses = newVirusBook()
	s.mines = newMineBook()
	s.recount = newRecount(borders.Landmasses(), table.Count()+1)
	s.played.Play(ground)
}

// A new epoch, and every stream cut: every client reads the map again, as after a restart.
func (s *Storage) restartLocked() {
	s.version = clicks.MapVersion{
		Epoch: max(uint32(s.clock.Now().Unix()), s.version.Epoch+1), //nolint:gosec // seconds fit a uint32 until 2106.
	}
	s.feed.cutAll()
}
