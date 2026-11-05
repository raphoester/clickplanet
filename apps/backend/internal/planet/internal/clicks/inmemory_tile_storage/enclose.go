package inmemory_tile_storage

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Enclose holds the lock through every write of `write`: one change at one version, then the fortifies it earned.
func (s *Storage) Enclose(_ context.Context, enclosure clicks.Enclosure, write func(tiles clicks.Claimable) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inside := &insideEnclosure{storage: s, enclosure: enclosure}
	err := write(inside)
	if len(inside.enclosure.Filled) > 0 {
		s.publishLocked(clicks.Change{Enclosure: &inside.enclosure})
	}
	for i := range inside.fortified {
		s.publishLocked(clicks.Change{Fortification: &inside.fortified[i]})
	}

	return err
}

type insideEnclosure struct {
	storage   *Storage
	enclosure clicks.Enclosure
	fortified []clicks.Fortification
}

var _ clicks.Claimable = (*insideEnclosure)(nil)

func (e *insideEnclosure) Owner(tile uint32) (string, bool) {
	if tile > e.storage.board.last() {
		return "", false
	}

	return e.storage.board.ownerOf(tile), true
}

func (e *insideEnclosure) Shields(tile uint32) int {
	if tile > e.storage.board.last() {
		return 0
	}

	return e.storage.board.shieldsOn(tile)
}

func (e *insideEnclosure) Strike(_ context.Context, tile uint32, owner string) bool {
	left, struck := e.storage.strikeLocked(tile, owner)
	if struck {
		e.enclosure.Filled = append(e.enclosure.Filled, tile)
		e.enclosure.Struck = append(e.enclosure.Struck, clicks.TileShields{Tile: tile, Shields: left})
	}

	return struck
}

func (e *insideEnclosure) Set(ctx context.Context, tile uint32, value string) error {
	return e.put(ctx, clicks.TileUpdate{Tile: tile, Value: value})
}

func (e *insideEnclosure) Click(ctx context.Context, tile uint32, value string) error {
	return e.put(ctx, clicks.TileUpdate{Tile: tile, Value: value, Clicked: true})
}

func (e *insideEnclosure) put(ctx context.Context, update clicks.TileUpdate) error {
	_, changed, err := e.storage.putLocked(update, holderOf(ctx))
	if changed {
		e.enclosure.Filled = append(e.enclosure.Filled, update.Tile)
	}

	return err
}

func (e *insideEnclosure) Fortify(ctx context.Context, tile uint32, flag string, most int) (clicks.Fortification, error) {
	fortification, err := e.storage.fortifyLocked(ctx, tile, flag, most)
	if err != nil {
		return clicks.Fortification{}, err
	}
	e.fortified = append(e.fortified, fortification)

	return fortification, nil
}
