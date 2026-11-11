package placements

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Tiles interface {
	Owner(tile uint32) (string, bool)
	Shields(tile uint32) int
}

type Countries interface {
	CheckCountry(country string) bool
}

type Allegiances interface {
	PlayError(ctx context.Context, payer clicks.Payer, flag string) error
}

func NewGate(tiles Tiles, countries Countries, allegiances Allegiances, rules func() charges.Rules) Gate {
	return Gate{tiles: tiles, countries: countries, allegiances: allegiances, rules: rules}
}

// Gate refuses a placement before a shield leaves the hand. The map checks the tile again when the shield comes.
type Gate struct {
	tiles       Tiles
	countries   Countries
	allegiances Allegiances
	rules       func() charges.Rules
}

type Aim struct {
	Tile    uint32
	Country string
	Most    int
}

func (g Gate) Aim(ctx context.Context, payer clicks.Payer, tile uint32, country string) (Aim, error) {
	if !g.countries.CheckCountry(country) {
		return Aim{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, country)
	}

	owner, ok := g.tiles.Owner(tile)
	if !ok {
		return Aim{}, fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, tile)
	}

	most := g.rules().Shield.PerTile
	if err := clicks.ShieldError(owner, country, g.tiles.Shields(tile), most); err != nil {
		return Aim{}, fmt.Errorf("tile %d: %w", tile, err)
	}

	if err := g.allegiances.PlayError(ctx, payer, country); err != nil && !errors.Is(err, allegiances.ErrUnread) {
		return Aim{}, err //nolint:wrapcheck // the book names the flag the caller plays for.
	}

	return Aim{Tile: tile, Country: country, Most: most}, nil
}
