package catalog

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

type Store interface {
	// Heard is the flag as the latest fact told it, and never heard for a flag no fact named.
	Heard(ctx context.Context, flag string) (Heard, error)
	// All is every flag a fact named, by flag.
	All(ctx context.Context) (map[string]Heard, error)
	// Change runs under a lock on the flag, and keeps what it answers when that differs from what it was handed.
	Change(ctx context.Context, flag string, change func(Heard) Heard) error
}

const readTimeout = time.Second

// lock is true while the rules in force let no account change its flag: a flag closed then stays playable until it
// lifts, so a battle plays to its end.
func NewCatalog(store Store, lock func() bool) Catalog {
	return Catalog{store: store, lock: lock}
}

// Catalog is the flags planet knows: the ones the game ships with, and the ones it heard opened or closed since
// (flags.v1.FlagOpened, FlagClosed), kept in its own schema.
type Catalog struct {
	store Store
	lock  func() bool
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
	return heard.playable() || (heard.known() && c.lock()), nil
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

// Move keeps the flag's pin: where it starts an arena, as a country starts on its capital.
func (c Catalog) Move(ctx context.Context, flag string, pin clicks.Pin, at time.Time) error {
	return c.change(ctx, flag, func(heard Heard) Heard { return heard.Moved(pin, at) })
}

// Pinned are the flags a player may play now that have no ground and stand on a pin: the regions the game ships with,
// and the flags heard open, by code. A flag closed under a lock is not one: a flag sits at an arena's start.
func (c Catalog) Pinned(ctx context.Context) ([]clicks.Pinned, error) {
	var pinned []clicks.Pinned
	for _, flag := range cpflags.Pins() {
		pinned = append(pinned, clicks.Pinned{Flag: flag.Code, Pin: clicks.Pin{Latitude: flag.Latitude, Longitude: flag.Longitude}})
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	heard, err := c.store.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the flags heard: %w", err)
	}
	for _, flag := range slices.Sorted(maps.Keys(heard)) {
		if pin, ok := heard[flag].seated(); ok {
			pinned = append(pinned, clicks.Pinned{Flag: flag, Pin: pin})
		}
	}
	return pinned, nil
}

func (c Catalog) change(ctx context.Context, flag string, change func(Heard) Heard) error {
	if err := c.store.Change(ctx, flag, change); err != nil {
		return fmt.Errorf("failed to keep what was heard of the flag %q: %w", flag, err)
	}
	return nil
}
