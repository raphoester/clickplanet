// Package spread_click is the spread bonus: while it runs, a click also takes
// every tile touching the one clicked.
//
// The server picks those tiles off its own map. A client that named them would
// be a client that could name any tiles it liked, which is the whole of the cheat.
package spread_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// Spreads says whether a caller holds a running spread bonus.
type Spreads interface {
	Spreading(scope string) bool
}

// Neighbours is the part of clicks.Geography this reads.
type Neighbours interface {
	Neighbours(id uint32) []uint32
}

type TileStorage interface {
	Set(ctx context.Context, tile uint32, value string) error
}

func New(implementation click.IUseCase, spreads Spreads, neighbours Neighbours, storage TileStorage) *UseCase {
	return &UseCase{
		implementation: implementation,
		spreads:        spreads,
		neighbours:     neighbours,
		storage:        storage,
	}
}

type UseCase struct {
	implementation click.IUseCase
	spreads        Spreads
	neighbours     Neighbours
	storage        TileStorage
}

// Execute spreads only a click the rule accepted, so a refused country or tile
// spreads nothing. The neighbours need no check of their own: the map only
// holds real tiles, and the country is the one the rule just accepted.
//
// A tile with no neighbours — one of the lone islands — takes itself and
// nothing else. That is what the map says, and the bonus does not pretend
// otherwise.
func (u *UseCase) Execute(ctx context.Context, in click.In) (click.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err != nil || !u.spreads.Spreading(cpctx.RateLimitKey(ctx)) {
		return out, err
	}

	for _, neighbour := range u.neighbours.Neighbours(in.TileID) {
		if err := u.storage.Set(ctx, neighbour, in.CountryID); err != nil {
			return out, fmt.Errorf("failed to spread onto tile %d: %w", neighbour, err)
		}
	}

	return out, nil
}
