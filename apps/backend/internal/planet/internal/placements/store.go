package placements

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Store keeps the placements under way. One that ended is no row, as an empty hand is none.
type Store interface {
	// Find locks each placement it finds until the transaction ends; one that ended is not found.
	Find(ctx context.Context, ids ...clicks.PlacementID) ([]Placement, error)
	All(ctx context.Context) ([]clicks.PlacementID, error)
	Save(ctx context.Context, placements ...Placement) error
}

type IDProvider interface {
	NewID() (clicks.PlacementID, error)
}
