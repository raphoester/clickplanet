package return_placements_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Placements interface {
	All(ctx context.Context) ([]clicks.PlacementID, error)
}

type Escrow interface {
	Return(ctx context.Context, ids ...clicks.PlacementID) error
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

func New(placements Placements, escrow Escrow) *UseCase {
	return &UseCase{placements: placements, escrow: escrow}
}

// UseCase runs at boot, before any request: every placement left was never saved on the map, since the flush that
// saves a shield ends its placement in the same transaction.
type UseCase struct {
	placements Placements
	escrow     Escrow
}

var _ Executor = (*UseCase)(nil)

// Execute answers how many placements it found.
func (u *UseCase) Execute(ctx context.Context) (int, error) {
	left, err := u.placements.All(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list the placements a stop left: %w", err)
	}

	return len(left), u.escrow.Return(ctx, left...) //nolint:wrapcheck // the escrow names each one.
}
