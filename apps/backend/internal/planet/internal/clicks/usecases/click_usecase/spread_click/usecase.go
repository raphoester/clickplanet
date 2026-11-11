package spread_click

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
)

type Spreads interface {
	Spend(ctx context.Context, holder charges.Holder, after func(charges.Held) (charges.Held, bool)) error
}

// Reach is the neighbours a spread of the flag takes: every one outside an arena.
type Reach interface {
	Around(flag string, tile uint32) []uint32
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

	if err := u.spreads.Spend(ctx, charges.HolderOf(clicks.PayerOf(ctx)), charges.Held.AfterSpreadClick); err != nil {
		if errors.Is(err, charges.ErrNotHeld) {
			return out, nil
		}
		return out, err
	}

	neighbours := u.reach.Around(in.CountryID, in.TileID)
	if err := u.spreader.Spread(ctx, in.TileID, in.CountryID, neighbours); err != nil {
		return out, fmt.Errorf("failed to spread from tile %d: %w", in.TileID, err)
	}

	// The neighbours are copied: Around hands back the map's own table.
	u.publisher.PublishSpread(notices.Spread{
		CountryID:  in.CountryID,
		Tile:       in.TileID,
		Neighbours: append([]uint32(nil), neighbours...),
	})

	return out, nil
}
