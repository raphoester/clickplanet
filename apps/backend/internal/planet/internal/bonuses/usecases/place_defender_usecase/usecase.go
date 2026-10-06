package place_defender_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var ErrNoDefender = errors.New("no defender to place")

type Defenders interface {
	SpendDefender(holder bonuses.Holder) bool
	Grant(holder bonuses.Holder, kind bonuses.Kind, amount int)
	Held(holder bonuses.Holder) bonuses.Held
}

type Tiles interface {
	Owner(tile uint32) (string, bool)
	Defenders(tile uint32) int
	Reinforce(ctx context.Context, tile uint32, country string, most int) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	TileID    uint32
	CountryID string

	Dud bool
}

func New(defenders Defenders, tiles Tiles, countries CountryChecker, perTile int) *UseCase {
	return &UseCase{defenders: defenders, tiles: tiles, countries: countries, perTile: perTile}
}

type UseCase struct {
	defenders Defenders
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

	if err := clicks.ReinforceError(owner, in.CountryID, u.tiles.Defenders(in.TileID), u.perTile); err != nil {
		return bonuses.Held{}, fmt.Errorf("tile %d: %w", in.TileID, err)
	}

	holder := bonuses.HolderOf(clicks.PayerOf(ctx))
	if !u.defenders.SpendDefender(holder) {
		return bonuses.Held{}, ErrNoDefender
	}

	if !in.Dud {
		if err := u.tiles.Reinforce(ctx, in.TileID, in.CountryID, u.perTile); err != nil {
			u.defenders.Grant(holder, bonuses.KindDefenders, 1)
			return bonuses.Held{}, fmt.Errorf("failed to place the defender: %w", err)
		}
	}

	return u.defenders.Held(holder), nil
}
