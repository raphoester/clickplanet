package schedule

import (
	"context"
	"errors"
	"time"
)

// Store keeps the operators' entries; the phases are the config's.
type Store interface {
	// Entry is ErrUnknownEntry when none has this id.
	Entry(ctx context.Context, id EntryID) (Entry, error)
	// Unended is every entry that ends after a time.
	Unended(ctx context.Context, at time.Time) ([]Entry, error)
	Save(ctx context.Context, entry Entry) error
	Delete(ctx context.Context, id EntryID) error
}

// Keeper is a module that plays the schedule's sets: planet, the antibot.
type Keeper interface {
	// Keeps is nil when the module keeps a set of this name, and an ErrUnknownSet that names the module when not.
	Keeps(ctx context.Context, set SetName) error
}

type IDProvider interface {
	NewID() (EntryID, error)
}

// Playable is nil when every module keeps the set, and names each that does not.
func Playable(ctx context.Context, set SetName, keepers ...Keeper) error {
	var errs []error
	for _, keeper := range keepers {
		if err := keeper.Keeps(ctx, set); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
