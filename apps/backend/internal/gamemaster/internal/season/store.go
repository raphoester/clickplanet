package season

import "context"

// Store keeps the seasons.
type Store interface {
	Calendar(ctx context.Context) (Calendar, error)
	Save(ctx context.Context, season Season) error
	Delete(ctx context.Context, number Number) error
}
