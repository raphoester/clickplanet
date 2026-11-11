package place_shield_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
)

var ErrNoShield = errors.New("no shield to place")

const returnTimeout = 5 * time.Second

type Gate interface {
	Aim(ctx context.Context, payer clicks.Payer, tile uint32, country string) (placements.Aim, error)
}

type Escrow interface {
	Take(ctx context.Context, holder charges.Holder) (charges.Held, clicks.PlacementID, error)
	Return(ctx context.Context, ids ...clicks.PlacementID) error
}

type Tiles interface {
	Shield(ctx context.Context, tile uint32, country string, most int, placement clicks.PlacementID) (int, error)
}

type In struct {
	TileID    uint32
	CountryID string
}

func New(gate Gate, escrow Escrow, tiles Tiles) *UseCase {
	return &UseCase{gate: gate, escrow: escrow, tiles: tiles}
}

type UseCase struct {
	gate   Gate
	escrow Escrow
	tiles  Tiles
}

func (u *UseCase) Execute(ctx context.Context, in In) (charges.Held, error) {
	payer := clicks.PayerOf(ctx)
	aim, err := u.gate.Aim(ctx, payer, in.TileID, in.CountryID)
	if err != nil {
		return charges.Held{}, err //nolint:wrapcheck // the gate names why.
	}

	held, placement, err := u.escrow.Take(ctx, charges.HolderOf(payer))
	if errors.Is(err, charges.ErrNotHeld) {
		return charges.Held{}, ErrNoShield
	}
	if err != nil {
		return charges.Held{}, err //nolint:wrapcheck // the escrow names it.
	}

	if _, err := u.tiles.Shield(ctx, aim.Tile, aim.Country, aim.Most, placement); err != nil {
		returning, cancel := context.WithTimeout(context.WithoutCancel(ctx), returnTimeout)
		defer cancel()
		return charges.Held{}, errors.Join(fmt.Errorf("failed to place the shield: %w", err), u.escrow.Return(returning, placement))
	}

	return held, nil
}
