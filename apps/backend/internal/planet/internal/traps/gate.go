// Package traps is a trap's way from the hand to the map, and what a trap sprung takes from its taker's bank: the rules
// it is laid by are clicks.Trap and clicks.Trapline, since the map keeps every trap with its tiles, and a take springs
// it in its own claim.
package traps

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

type Allegiances interface {
	PlayError(ctx context.Context, payer clicks.Payer, flag string) error
}

type Refusals interface {
	Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error)
}

func NewGate(tiles Tiles, countries clicks.Flags, allegiances Allegiances, refusals Refusals) Gate {
	return Gate{tiles: tiles, countries: countries, allegiances: allegiances, refusals: refusals}
}

// Gate refuses a trap before it leaves the hand: a banned caller, a flag it does not play for, and a tile not the
// flag's. A capital takes one: the take that knocks its flag out springs it first. The map checks the tile again under
// its lock.
type Gate struct {
	tiles       Tiles
	countries   clicks.Flags
	allegiances Allegiances
	refusals    Refusals
}

// A failed read of the refusals or of the flag refuses nothing, as for a click.
func (g Gate) Lay(ctx context.Context, payer clicks.Payer, tile uint32, country string) error {
	if refusal, refused, err := g.refusals.Refusal(ctx, payer); err == nil && refused {
		return refusal
	}
	if err := clicks.CheckFlag(ctx, g.countries, country); err != nil {
		return err //nolint:wrapcheck // the check names it.
	}
	owner, ok := g.tiles.Owner(tile)
	if !ok {
		return fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, tile)
	}
	if err := g.allegiances.PlayError(ctx, payer, country); err != nil && !errors.Is(err, allegiances.ErrUnread) {
		return err //nolint:wrapcheck // the book names the flag the caller plays for.
	}
	if owner != country {
		return fmt.Errorf("tile %d: %w", tile, clicks.ErrNotTrappable)
	}

	return nil
}
