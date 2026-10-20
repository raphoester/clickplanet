package clear_area_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Map interface {
	Owner(tile uint32) (string, bool)
	Restore(ctx context.Context, restorations []clicks.Restoration) (int, error)
}

type Areas interface {
	Around(tile uint32, rings uint32) ([]uint32, error)
}

type In struct {
	Tile   uint32
	Rings  uint32
	DryRun bool
}

type Out struct {
	Tiles   int
	Owned   int
	Cleared int
}

func New(tiles Map, areas Areas, pace clicks.Pacing) *UseCase {
	return &UseCase{tiles: tiles, areas: areas, pacing: pace}
}

type UseCase struct {
	tiles  Map
	areas  Areas
	pacing clicks.Pacing
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	if u.pacing.Batch <= 0 {
		return Out{}, errors.New("clear batch must be positive")
	}

	area, err := u.areas.Around(in.Tile, in.Rings)
	if err != nil {
		return Out{}, fmt.Errorf("cannot clear: %w", err)
	}

	restorations := clicks.Emptying(area, u.tiles)
	out := Out{Tiles: len(area), Owned: len(restorations)}

	if in.DryRun {
		return out, nil
	}

	for len(restorations) > 0 {
		batch := restorations[:min(u.pacing.Batch, len(restorations))]
		restorations = restorations[len(batch):]

		cleared, err := u.tiles.Restore(ctx, batch)
		out.Cleared += cleared
		if err != nil {
			return out, fmt.Errorf("failed to clear tiles: %w", err)
		}

		if len(restorations) > 0 {
			if err := u.pacing.Wait(ctx); err != nil {
				return out, fmt.Errorf("clear %w", err)
			}
		}
	}

	return out, nil
}
