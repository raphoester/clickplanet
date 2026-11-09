package inmemory_tile_storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var errWorldIsNoArena = errors.New("the world's map is no arena's")

// Blank plays the blank map of an arena once keep says to: what changed is saved to the map in play first, which
// stays as it is, and the arena's map has no tile yet. The map starts a new epoch and every stream is cut.
func (s *Storage) Blank(ctx context.Context, arena clicks.MapID, keep func(ctx context.Context) (bool, error)) (bool, error) {
	if arena == clicks.World {
		return false, errWorldIsNoArena
	}

	s.flushing.Lock()
	defer s.flushing.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.flushLocked(ctx); err != nil {
		return false, fmt.Errorf("failed to save the map before blanking it: %w", err)
	}
	blanked, err := keep(ctx)
	if err != nil || !blanked {
		return false, err
	}

	s.playing = arena
	s.emptyLocked()
	s.restartLocked()

	return true, nil
}

// PutBack deletes an arena's map once keep says to, in keep's transaction. When the arena's map plays, the world's
// plays again, read as the boot reads it: a new epoch and every stream cut. What memory held of the arena is dropped.
func (s *Storage) PutBack(ctx context.Context, arena clicks.MapID, keep func(ctx context.Context) (bool, error)) (bool, error) {
	if arena == clicks.World {
		return false, errWorldIsNoArena
	}

	s.flushing.Lock()
	defer s.flushing.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	dropped, err := s.persistence.Wipe(ctx, arena, keep)
	if err != nil {
		return false, fmt.Errorf("failed to delete the arena's map: %w", err)
	}
	if !dropped || s.playing != arena {
		return dropped, nil
	}

	s.playing = clicks.World
	s.emptyLocked()
	defer s.restartLocked()
	if err := s.fillLocked(ctx); err != nil {
		return true, fmt.Errorf("deleted the arena's map, but could not read the world's: %w", err)
	}

	return true, nil
}
