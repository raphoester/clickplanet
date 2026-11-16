// Package mines is a mine's way from the hand to the map, and the runner that sets off the mines a take tripped: the
// rules they play by are clicks.Mine and clicks.Minefield, since the map keeps every mine with its tiles.
package mines

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Tiles interface {
	Owner(tile uint32) (string, bool)
}

type Capitals interface {
	KnocksOut(tile uint32) bool
}

type Countries interface {
	CheckCountry(country string) bool
}

type Allegiances interface {
	PlayError(ctx context.Context, payer clicks.Payer, flag string) error
}

type Refusals interface {
	Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error)
}

func NewGate(tiles Tiles, capitals Capitals, countries Countries, allegiances Allegiances, refusals Refusals) Gate {
	return Gate{tiles: tiles, capitals: capitals, countries: countries, allegiances: allegiances, refusals: refusals}
}

// Gate refuses a mine before it leaves the hand: a banned caller, a flag it does not play for, a tile not the flag's,
// and a capital whose take knocks its flag out, which would clear the mine with the flag's land before it went off. The
// map checks the tile again under its lock.
type Gate struct {
	tiles       Tiles
	capitals    Capitals
	countries   Countries
	allegiances Allegiances
	refusals    Refusals
}

// A failed read of the refusals or of the flag refuses nothing, as for a click.
func (g Gate) Lay(ctx context.Context, payer clicks.Payer, tile uint32, country string) error {
	if refusal, refused, err := g.refusals.Refusal(ctx, payer); err == nil && refused {
		return refusal
	}
	if !g.countries.CheckCountry(country) {
		return fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, country)
	}
	owner, ok := g.tiles.Owner(tile)
	if !ok {
		return fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, tile)
	}
	if err := g.allegiances.PlayError(ctx, payer, country); err != nil && !errors.Is(err, allegiances.ErrUnread) {
		return err //nolint:wrapcheck // the book names the flag the caller plays for.
	}
	if owner != country {
		return fmt.Errorf("tile %d: %w", tile, clicks.ErrNotMineable)
	}
	if g.capitals.KnocksOut(tile) {
		return fmt.Errorf("tile %d is a capital the knock-out plays on: %w", tile, clicks.ErrNotMineable)
	}

	return nil
}
