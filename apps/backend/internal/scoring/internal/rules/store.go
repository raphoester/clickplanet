package rules

import "context"

type Store interface {
	// Set is the set of this name: ErrUnknownSet when none is kept, ErrInvalidSet when the one kept cannot play.
	Set(ctx context.Context, name SetName) (Set, error)
	Save(ctx context.Context, set Set) error
}
