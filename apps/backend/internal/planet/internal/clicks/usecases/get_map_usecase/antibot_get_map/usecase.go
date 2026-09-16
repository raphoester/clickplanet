// Package antibot_get_map tells the guard how much of the map each caller reads.
package antibot_get_map

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in get_map_usecase.In) (clicks.DenseBatch, error)
}

type FetchGuard interface {
	Fetched(scope string, maps float64, offMap bool)
}

type MaxIndexReader interface {
	MaxIndex() uint32
}

func New(implementation UseCase, guard FetchGuard, board MaxIndexReader) *Decorator {
	return &Decorator{implementation: implementation, guard: guard, board: board}
}

type Decorator struct {
	implementation UseCase
	guard          FetchGuard
	board          MaxIndexReader
}

func (d *Decorator) Execute(ctx context.Context, in get_map_usecase.In) (clicks.DenseBatch, error) {
	batch, err := d.implementation.Execute(ctx, in)
	if err != nil {
		return clicks.DenseBatch{}, fmt.Errorf("failed to read the map: %w", err)
	}

	// Two bytes per tile: what was read, whatever the request asked for.
	maxIndex := d.board.MaxIndex()
	d.guard.Fetched(cpctx.RateLimitKey(ctx), float64(len(batch.Tiles)/2)/float64(maxIndex), offMap(in, maxIndex))

	return batch, nil
}

// Tile ids run from 1 and the web app clamps its last batch, so neither bound can fall
// outside the map whatever its batch size. The bots of 2026-09-16 walked from 0 and past the end.
func offMap(in get_map_usecase.In, maxIndex uint32) bool {
	end := in.End
	if end == 0 {
		end = maxIndex
	}

	return in.Start == 0 || end > maxIndex
}
