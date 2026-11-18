package inmemory_tile_storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var errWorldIsNoArena = errors.New("the world's map is no arena's")

// Blank plays the blank map of an arena on board once keep says to: what changed is saved to the map in play first,
// which stays as it is, and the arena's map holds only its start, stored in keep's transaction. keep is told the arena's
// stage. The map starts a new epoch and every stream is cut, and every rule reads board's ground from then on.
func (s *Storage) Blank(
	ctx context.Context, arena clicks.Stage, start []clicks.StartTile, keep func(ctx context.Context, on clicks.Stage) (bool, error),
) (bool, error) {
	if arena.Map == clicks.World {
		return false, errWorldIsNoArena
	}
	ground, err := s.grounds.GroundOf(arena.Board)
	if err != nil {
		return false, fmt.Errorf("failed to read the ground of the arena's board: %w", err)
	}

	s.staging.Lock()
	defer s.staging.Unlock()

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
	blanked, err := s.persistence.Open(ctx, arena, tiles, func(ctx context.Context) (bool, error) { return keep(ctx, arena) })
	if err != nil || !blanked {
		return false, err //nolint:wrapcheck // keep and the store name it.
	}

	s.stage = arena
	s.groundLocked(ground)
	defer s.restartLocked()
	if err := s.fillLocked(ctx); err != nil {
		return true, fmt.Errorf("opened the arena, but could not read its map: %w", err)
	}

	return true, nil
}

// PutBack deletes an arena's map once keep says to, in keep's transaction. When the arena's map plays, the world plays
// again on world, the board of its era, read as the boot reads it: a new epoch, every stream cut, and every rule reading
// that board's ground. What changed is saved first, so every shield placed is confirmed; then what memory held of the
// arena is dropped.
func (s *Storage) PutBack(
	ctx context.Context, arena clicks.MapID, world clicks.Stage, keep func(ctx context.Context) (bool, error),
) (bool, error) {
	if arena == clicks.World || world.Map != clicks.World {
		return false, errWorldIsNoArena
	}
	ground, err := s.grounds.GroundOf(world.Board)
	if err != nil {
		return false, fmt.Errorf("failed to read the ground of the world's board: %w", err)
	}

	s.staging.Lock()
	defer s.staging.Unlock()

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
	if !dropped || s.stage.Map != arena {
		return dropped, nil
	}

	s.stage = world
	s.groundLocked(ground)
	defer s.restartLocked()
	if err := s.fillLocked(ctx); err != nil {
		return true, fmt.Errorf("deleted the arena's map, but could not read the world's: %w", err)
	}

	return true, nil
}
