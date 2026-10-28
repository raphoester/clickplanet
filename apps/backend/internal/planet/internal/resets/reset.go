package resets

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

var ErrInvalidReset = errors.New("this reset cannot be applied")

// ID names a reset as the module that asks for it names it: season-1.
type ID string

const maxID = 64

// Reset wipes the map once: every tile blank, with no holder and no shield, and no fortify lock.
type Reset struct {
	ID ID
	At time.Time
}

func New(id string, at time.Time) (Reset, error) {
	if id == "" || utf8.RuneCountInString(id) > maxID {
		return Reset{}, fmt.Errorf("%w: its id %q is empty or longer than %d", ErrInvalidReset, id, maxID)
	}
	if at.IsZero() {
		return Reset{}, fmt.Errorf("%w: %q has no time", ErrInvalidReset, id)
	}
	return Reset{ID: ID(id), At: at.UTC()}, nil
}

// Store keeps each reset applied.
type Store interface {
	// Keep is true for a reset not kept yet, and keeps it; one kept already is false and changes nothing.
	Keep(ctx context.Context, reset Reset) (bool, error)
	// After is every reset due after a time, by time.
	After(ctx context.Context, at time.Time) ([]Reset, error)
}
