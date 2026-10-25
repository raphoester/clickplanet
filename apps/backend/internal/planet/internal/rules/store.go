package rules

import "context"

type Store interface {
	// Set is the set of this name, or ErrUnknownSet.
	Set(ctx context.Context, name SetName) (Set, error)
	Save(ctx context.Context, set Set) error
}
