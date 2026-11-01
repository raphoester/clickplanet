package rules

import "context"

type Store interface {
	// Set is the set of this name over default: ErrUnknownSet when either is not kept, ErrInvalidSet when it cannot play.
	Set(ctx context.Context, name SetName) (Set, error)
	Names(ctx context.Context) ([]SetName, error)
	Save(ctx context.Context, set Set) error
}
