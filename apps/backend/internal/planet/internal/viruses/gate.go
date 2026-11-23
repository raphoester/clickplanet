// Package viruses is a virus's way from the hand to the map, and the runner that spreads it: the rules it plays by are
// clicks.Virus and clicks.Contagion, since the map keeps every live virus with its tiles.
package viruses

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Tiles interface {
	Owner(tile uint32) (string, bool)
	Shields(tile uint32) int
}

type Allegiances interface {
	PlayError(ctx context.Context, payer clicks.Payer, flag string) error
}

type Reach interface {
	Refuses(flag string, tile uint32) error
}

type Refusals interface {
	Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error)
}

func NewGate(tiles Tiles, countries clicks.Flags, allegiances Allegiances, reach Reach, refusals Refusals) Gate {
	return Gate{tiles: tiles, countries: countries, allegiances: allegiances, reach: reach, refusals: refusals}
}

// Gate refuses a virus before it leaves the hand: a banned caller, a flag it does not play for, a tile out of its reach,
// and a tile of no other flag or with a shield. The map checks the tile again under its lock.
type Gate struct {
	tiles       Tiles
	countries   clicks.Flags
	allegiances Allegiances
	reach       Reach
	refusals    Refusals
}

// A failed read of the refusals or of the flag refuses nothing, as for a click.
func (g Gate) Aim(ctx context.Context, payer clicks.Payer, tile uint32, country string) error {
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
	if err := g.reach.Refuses(country, tile); err != nil {
		return err //nolint:wrapcheck // the reach names the tile and the flag.
	}
	if owner == "" || owner == country || g.tiles.Shields(tile) > 0 {
		return fmt.Errorf("tile %d: %w", tile, clicks.ErrNotInfectable)
	}

	return nil
}
