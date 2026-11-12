package spread_click

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
)

const returnTimeout = 5 * time.Second

type Spreads interface {
	TakeSpread(ctx context.Context, holder charges.Holder) (charges.Spread, error)
	GiveBackSpread(ctx context.Context, holder charges.Holder, spread charges.Spread) error
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
	if !in.Spread {
		return u.implementation.Execute(ctx, in)
	}

	holder := charges.HolderOf(clicks.PayerOf(ctx))
	spread, err := u.spreads.TakeSpread(ctx, holder)
	if err != nil {
		return click_usecase.Out{}, fmt.Errorf("failed to take a spread click: %w", err)
	}
	in.Rings = spread.Rings

	out, err := u.implementation.Execute(ctx, in)
	if err != nil {
		if back := u.giveBack(ctx, holder, spread); back != nil {
			return out, errors.Join(err, back)
		}
		return out, err
	}
	if in.Rings == 0 {
		return out, nil
	}

	tiles := u.reach.Around(in.CountryID, in.TileID, in.Rings)
	if err := u.spreader.Spread(ctx, in.TileID, in.CountryID, tiles); err != nil {
		return out, fmt.Errorf("failed to spread from tile %d: %w", in.TileID, err)
	}
	u.publisher.PublishSpread(notices.Spread{CountryID: in.CountryID, Tile: in.TileID, Neighbours: tiles})

	return out, nil
}

func (u *UseCase) giveBack(ctx context.Context, holder charges.Holder, spread charges.Spread) error {
	returning, cancel := context.WithTimeout(context.WithoutCancel(ctx), returnTimeout)
	defer cancel()

	if err := u.spreads.GiveBackSpread(returning, holder, spread); err != nil {
		return fmt.Errorf("failed to give back the spread click of a refused click: %w", err)
	}

	return nil
}
