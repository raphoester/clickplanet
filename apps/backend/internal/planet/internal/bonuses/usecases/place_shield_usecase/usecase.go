package place_shield_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var ErrNoShield = errors.New("no shield to place")

type Shields interface {
	SpendShield(holder bonuses.Holder) bool
	Grant(holder bonuses.Holder, kind bonuses.Kind, amount int)
	Held(holder bonuses.Holder) bonuses.Held
}

type Tiles interface {
	Owner(tile uint32) (string, bool)
	Shields(tile uint32) int
	Shield(ctx context.Context, tile uint32, country string, most int) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	TileID    uint32
	CountryID string

	Dud bool
}

func New(shields Shields, tiles Tiles, countries CountryChecker, perTile int) *UseCase {
	return &UseCase{shields: shields, tiles: tiles, countries: countries, perTile: perTile}
}

type UseCase struct {
	shields   Shields
	tiles     Tiles
	countries CountryChecker
	perTile   int
}

func (u *UseCase) Execute(ctx context.Context, in In) (bonuses.Held, error) {
	if !u.countries.CheckCountry(in.CountryID) {
		return bonuses.Held{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, in.CountryID)
	}

	owner, ok := u.tiles.Owner(in.TileID)
	if !ok {
		return bonuses.Held{}, fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, in.TileID)
	}

	if err := clicks.ShieldError(owner, in.CountryID, u.tiles.Shields(in.TileID), u.perTile); err != nil {
		return bonuses.Held{}, fmt.Errorf("tile %d: %w", in.TileID, err)
	}

	holder := bonuses.HolderOf(clicks.PayerOf(ctx))
	if !u.shields.SpendShield(holder) {
		return bonuses.Held{}, ErrNoShield
	}

	if !in.Dud {
		if err := u.tiles.Shield(ctx, in.TileID, in.CountryID, u.perTile); err != nil {
			u.shields.Grant(holder, bonuses.KindShields, 1)
			return bonuses.Held{}, fmt.Errorf("failed to place the shield: %w", err)
		}
	}

	return u.shields.Held(holder), nil
}
