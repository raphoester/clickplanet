package inmemory_tile_storage

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Enclose holds the lock through every write of `write`: one change at one version, then the fortifies it earned, then
// the knock-outs.
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
	for i := range inside.knockedOut {
		s.publishLocked(clicks.Change{KnockOut: &inside.knockedOut[i]})
	}

	return err
}

type insideEnclosure struct {
	storage    *Storage
	enclosure  clicks.Enclosure
	fortified  []clicks.Fortification
	knockedOut []clicks.KnockOut
}

var _ clicks.Claimable = (*insideEnclosure)(nil)

func (e *insideEnclosure) Tile(id uint32) (clicks.Tile, error) {
	return e.storage.tileLocked(id)
}

func (e *insideEnclosure) Set(_ context.Context, tile clicks.Tile) error {
	return e.put(tile)
}

func (e *insideEnclosure) Click(_ context.Context, tile clicks.Tile) error {
	return e.put(tile)
}

func (e *insideEnclosure) put(tile clicks.Tile) error {
	update, changed, err := e.storage.writeLocked(tile, false)
	if !changed {
		return err
	}
	e.enclosure.Filled = append(e.enclosure.Filled, update.Tile)
	if update.Value == update.Previous {
		e.enclosure.Struck = append(e.enclosure.Struck, clicks.TileShields{Tile: update.Tile, Shields: update.Shields})
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

func (e *insideEnclosure) KnockOut(ctx context.Context, tile uint32, flag string) (clicks.KnockOut, error) {
	knockOut, err := e.storage.knockOutLocked(ctx, tile, flag)
	if err != nil {
		return clicks.KnockOut{}, err
	}
	e.knockedOut = append(e.knockedOut, knockOut)

	return knockOut, nil
}
