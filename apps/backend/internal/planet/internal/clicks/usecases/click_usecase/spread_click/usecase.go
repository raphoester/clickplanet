package spread_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Spreads interface {
	SpendSpreadClick(holder bonuses.Holder) bool
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
	if err != nil || !in.Spread || !u.spreads.SpendSpreadClick(bonuses.HolderOf(clicks.PayerOf(ctx))) {
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
