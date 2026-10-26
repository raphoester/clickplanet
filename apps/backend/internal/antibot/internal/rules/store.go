package rules

import "context"

type Store interface {
	// Set is the set of this name, or ErrUnknownSet.
	Set(ctx context.Context, name SetName) (Set, error)
	// Save makes the set, or replaces the values of the set of its name.
	Save(ctx context.Context, set Set) error
}
