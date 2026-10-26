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
	Spend(ctx context.Context, holder bonuses.Holder, after func(bonuses.Held) (bonuses.Held, bool)) error
	Grant(ctx context.Context, holder bonuses.Holder, kind bonuses.Kind, amount int) error
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

type Tiles interface {
	Owner(tile uint32) (string, bool)
	Shields(tile uint32) int
	Shield(ctx context.Context, tile uint32, country string, most int) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type Allegiances interface {
	PlayError(ctx context.Context, payer clicks.Payer, flag string) error
}

type In struct {
	TileID    uint32
	CountryID string
}

func New(shields Shields, tiles Tiles, countries CountryChecker, allegiances Allegiances, rules func() bonuses.Rules) *UseCase {
	return &UseCase{shields: shields, tiles: tiles, countries: countries, allegiances: allegiances, rules: rules}
}

// A tile above a smaller per-tile cap keeps its shields, and takes none until it is under it.
type UseCase struct {
	shields     Shields
	tiles       Tiles
	countries   CountryChecker
	allegiances Allegiances
	rules       func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, in In) (bonuses.Held, error) {
	if !u.countries.CheckCountry(in.CountryID) {
		return bonuses.Held{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, in.CountryID)
	}

	owner, ok := u.tiles.Owner(in.TileID)
	if !ok {
		return bonuses.Held{}, fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, in.TileID)
	}

	perTile := u.rules().Shield.PerTile
	if err := clicks.ShieldError(owner, in.CountryID, u.tiles.Shields(in.TileID), perTile); err != nil {
		return bonuses.Held{}, fmt.Errorf("tile %d: %w", in.TileID, err)
	}

	payer := clicks.PayerOf(ctx)
	if err := u.allegiances.PlayError(ctx, payer, in.CountryID); err != nil {
		return bonuses.Held{}, err
	}

	holder := bonuses.HolderOf(payer)
	if err := u.shields.Spend(ctx, holder, bonuses.Held.AfterShield); err != nil {
		if errors.Is(err, bonuses.ErrNotHeld) {
			return bonuses.Held{}, ErrNoShield
		}
		return bonuses.Held{}, err
	}

	if err := u.tiles.Shield(ctx, in.TileID, in.CountryID, perTile); err != nil {
		return bonuses.Held{}, errors.Join(fmt.Errorf("failed to place the shield: %w", err),
			u.shields.Grant(ctx, holder, bonuses.KindShields, 1))
	}

	return u.shields.Held(ctx, holder)
}
