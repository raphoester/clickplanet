package spread_click

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Spreads interface {
	Spend(ctx context.Context, holder bonuses.Holder, after func(bonuses.Held) (bonuses.Held, bool)) error
}

type Neighbours interface {
	Neighbours(id uint32) []uint32
}

type Spreader interface {
	Spread(ctx context.Context, tile uint32, flag string, neighbours []uint32) error
}

type Publisher interface {
	PublishSpread(spread bonuses.Spread)
}

func New(
	implementation click_usecase.IUseCase,
	spreads Spreads,
	neighbours Neighbours,
	spreader Spreader,
	publisher Publisher,
) *UseCase {
	return &UseCase{
		implementation: implementation,
		spreads:        spreads,
		neighbours:     neighbours,
		spreader:       spreader,
		publisher:      publisher,
	}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	spreads        Spreads
	neighbours     Neighbours
	spreader       Spreader
	publisher      Publisher
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err != nil || !in.Spread {
		return out, err
	}

	if err := u.spreads.Spend(ctx, bonuses.HolderOf(clicks.PayerOf(ctx)), bonuses.Held.AfterSpreadClick); err != nil {
		if errors.Is(err, bonuses.ErrNotHeld) {
			return out, nil
		}
		return out, err
	}

	neighbours := u.neighbours.Neighbours(in.TileID)
	if err := u.spreader.Spread(ctx, in.TileID, in.CountryID, neighbours); err != nil {
		return out, fmt.Errorf("failed to spread from tile %d: %w", in.TileID, err)
	}

	// The neighbours are copied: Neighbours hands back the map's own table.
	u.publisher.PublishSpread(bonuses.Spread{
		CountryID:  in.CountryID,
		Tile:       in.TileID,
		Neighbours: append([]uint32(nil), neighbours...),
	})

	return out, nil
}
