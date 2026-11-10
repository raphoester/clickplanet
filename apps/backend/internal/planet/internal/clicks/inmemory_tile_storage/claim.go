package inmemory_tile_storage

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Claim holds the lock from the first read of `write` to its last write, each published as it lands.
func (s *Storage) Claim(_ context.Context, write func(tiles clicks.Claimable) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return write(&insideClaim{storage: s})
}

type insideClaim struct {
	storage *Storage
}

var _ clicks.Claimable = (*insideClaim)(nil)

func (c *insideClaim) Tile(id uint32) (clicks.Tile, error) {
	return c.storage.tileLocked(id)
}

func (c *insideClaim) Set(_ context.Context, tile clicks.Tile) error {
	return c.put(tile, false)
}

func (c *insideClaim) Click(_ context.Context, tile clicks.Tile) error {
	return c.put(tile, true)
}

func (c *insideClaim) put(tile clicks.Tile, clicked bool) error {
	update, changed, err := c.storage.writeLocked(tile, clicked)
	if changed {
		c.storage.publishLocked(clicks.Change{Update: &update})
	}

	return err
}

func (c *insideClaim) Fortify(ctx context.Context, tile uint32, flag string, most int) (clicks.Fortification, error) {
	fortification, err := c.storage.fortifyLocked(ctx, tile, flag, most)
	if err != nil {
		return clicks.Fortification{}, err
	}
	c.storage.publishLocked(clicks.Change{Fortification: &fortification})

	return fortification, nil
}

func (c *insideClaim) KnockOut(ctx context.Context, tile uint32, flag string) (clicks.KnockOut, error) {
	knockOut, err := c.storage.knockOutLocked(ctx, tile, flag)
	if err != nil {
		return clicks.KnockOut{}, err
	}
	c.storage.publishLocked(clicks.Change{KnockOut: &knockOut})

	return knockOut, nil
}
