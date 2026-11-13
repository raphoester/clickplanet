package spread_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
)

type Spreads interface {
	TakeSpread(ctx context.Context, holder charges.Holder) (int, error)
}

type Reach interface {
	Around(flag string, tile uint32, rings int) []uint32
}

type Spreader interface {
	Spread(ctx context.Context, tile uint32, flag string, neighbours []uint32) error
}

type Publisher interface {
	PublishSpread(spread notices.Spread)
}

func New(
	implementation click_usecase.IUseCase,
	spreads Spreads,
	reach Reach,
	spreader Spreader,
	publisher Publisher,
) *UseCase {
	return &UseCase{
		implementation: implementation,
		spreads:        spreads,
		reach:          reach,
		spreader:       spreader,
		publisher:      publisher,
	}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	spreads        Spreads
	reach          Reach
	spreader       Spreader
	publisher      Publisher
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err != nil || !in.Spread {
		return out, err
	}

	rings, err := u.spreads.TakeSpread(ctx, charges.HolderOf(clicks.PayerOf(ctx)))
	if err != nil {
		return out, fmt.Errorf("failed to take a spread click: %w", err)
	}
	if rings == 0 {
		return out, nil
	}

	tiles := u.reach.Around(in.CountryID, in.TileID, rings)
	if err := u.spreader.Spread(ctx, in.TileID, in.CountryID, tiles); err != nil {
		return out, fmt.Errorf("failed to spread from tile %d: %w", in.TileID, err)
	}
	u.publisher.PublishSpread(notices.Spread{CountryID: in.CountryID, Tile: in.TileID, Neighbours: tiles})

	return out, nil
}
