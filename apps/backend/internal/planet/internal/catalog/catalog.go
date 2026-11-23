package catalog

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

type Store interface {
	// Heard is the flag as the latest fact told it, and never heard for a flag no fact named.
	Heard(ctx context.Context, flag string) (Heard, error)
	// Change runs under a lock on the flag, and keeps what it answers when that differs from what it was handed.
	Change(ctx context.Context, flag string, change func(Heard) Heard) error
}

const readTimeout = time.Second

func NewCatalog(store Store) Catalog {
	return Catalog{store: store}
}

// Catalog is the flags planet knows: the ones the game ships with, and the ones it heard opened or closed since
// (flags.v1.FlagOpened, FlagClosed), kept in its own schema.
type Catalog struct {
	store Store
}

// A player may play for a flag the game ships with, or for one opened and not closed since.
func (c Catalog) Playable(ctx context.Context, flag string) (bool, error) {
	if cpflags.IsBuiltIn(flag) {
		return true, nil
	}
	heard, err := c.heard(ctx, flag)
	if err != nil {
		return false, err
	}
	return heard.playable(), nil
}

// A closed flag is known: history still names it.
func (c Catalog) Known(ctx context.Context, flag string) (bool, error) {
	if cpflags.IsBuiltIn(flag) {
		return true, nil
	}
	heard, err := c.heard(ctx, flag)
	if err != nil {
		return false, err
	}
	return heard.known(), nil
}

func (c Catalog) heard(ctx context.Context, flag string) (Heard, error) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	heard, err := c.store.Heard(ctx, flag)
	if err != nil {
		return Heard{}, fmt.Errorf("failed to read what was heard of the flag %q: %w", flag, err)
	}
	return heard, nil
}

func (c Catalog) Open(ctx context.Context, flag string, at time.Time) error {
	return c.change(ctx, flag, func(heard Heard) Heard { return heard.Opened(at) })
}

func (c Catalog) Close(ctx context.Context, flag string, at time.Time) error {
	return c.change(ctx, flag, func(heard Heard) Heard { return heard.Closed(at) })
}

func (c Catalog) change(ctx context.Context, flag string, change func(Heard) Heard) error {
	if err := c.store.Change(ctx, flag, change); err != nil {
		return fmt.Errorf("failed to keep what was heard of the flag %q: %w", flag, err)
	}
	return nil
}
