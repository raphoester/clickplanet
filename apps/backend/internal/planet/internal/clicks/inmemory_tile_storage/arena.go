package inmemory_tile_storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var errWorldIsNoArena = errors.New("the world's map is no arena's")

// Blank plays the blank map of an arena once keep says to: what changed is saved to the map in play first, which
// stays as it is, and the arena's map holds only its start, stored in keep's transaction. The map starts a new epoch
// and every stream is cut.
func (s *Storage) Blank(
	ctx context.Context, arena clicks.MapID, start []clicks.StartTile, keep func(ctx context.Context) (bool, error),
) (bool, error) {
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
	tiles := make([]Tile, 0, len(start))
	for _, seed := range start {
		tiles = append(tiles, Tile{ID: seed.Tile, Owner: seed.Country, Shields: seed.Shields})
	}
	blanked, err := s.persistence.Open(ctx, arena, tiles, keep)
	if err != nil || !blanked {
		return false, err //nolint:wrapcheck // keep and the store name it.
	}

	s.playing = arena
	s.emptyLocked()
	defer s.restartLocked()
	if err := s.fillLocked(ctx); err != nil {
		return true, fmt.Errorf("opened the arena, but could not read its map: %w", err)
	}

	return true, nil
}

// PutBack deletes an arena's map once keep says to, in keep's transaction. When the arena's map plays, the world's
// plays again, read as the boot reads it: a new epoch and every stream cut. What changed is saved first, so every
// shield placed is confirmed; then what memory held of the arena is dropped.
func (s *Storage) PutBack(ctx context.Context, arena clicks.MapID, keep func(ctx context.Context) (bool, error)) (bool, error) {
	if arena == clicks.World {
		return false, errWorldIsNoArena
	}

	s.flushing.Lock()
	defer s.flushing.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.flushLocked(ctx); err != nil {
		return false, fmt.Errorf("failed to save the map before deleting the arena's: %w", err)
	}
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
