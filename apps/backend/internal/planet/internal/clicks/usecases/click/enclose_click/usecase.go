// Package enclose_click is the enclose bonus: while it runs, a click that closes
// a shape of the caller's own tiles also takes every tile inside it.
//
// The server finds the shape off its own map, the way spread_click finds the
// neighbours. A client that named the tiles inside would be a client that could
// name any tiles it liked.
package enclose_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// Enclosures says whether a caller holds a running enclose bonus, and spends it.
type Enclosures interface {
	Enclosing(scope string) (maxTiles int, ok bool)
	Spend(scope string) (left int, ok bool)
}

// Neighbours is the part of clicks.Geography this reads.
type Neighbours interface {
	Neighbours(id uint32) []uint32
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

type TileStorage interface {
	Set(ctx context.Context, tile uint32, value string) error
}

// Publisher tells the planet a shape was closed, so every client can show it.
type Publisher interface {
	PublishEnclosed(scope string, enclosed bonus.Enclosed)
}

func New(
	implementation click.IUseCase,
	enclosures Enclosures,
	neighbours Neighbours,
	owners Owners,
	storage TileStorage,
	publisher Publisher,
) *UseCase {
	return &UseCase{
		implementation: implementation,
		enclosures:     enclosures,
		neighbours:     neighbours,
		owners:         owners,
		storage:        storage,
		publisher:      publisher,
	}
}

type UseCase struct {
	implementation click.IUseCase
	enclosures     Enclosures
	neighbours     Neighbours
	owners         Owners
	storage        TileStorage
	publisher      Publisher
}

// Execute looks for shapes only around a click the rule accepted, so a refused
// country or tile closes nothing.
//
// Only a click that takes a tile closes a shape. A click on a tile the caller
// already held changes nothing on the map, so it closes nothing either: a shape
// finished before the bonus stays as it is, and the bonus is for closing one.
//
// Each shape found costs one of the bonus's shapes, and the search stops when
// they run out: a click that closes two shapes with one left takes the first.
// A shape too big, or open, costs nothing — so neither does a triangle of three
// tiles, which has no inside.
func (u *UseCase) Execute(ctx context.Context, in click.In) (click.Out, error) {
	scope := cpctx.RateLimitKey(ctx)

	maxTiles, enclosing := u.enclosures.Enclosing(scope)
	if !enclosing {
		return u.implementation.Execute(ctx, in)
	}

	// Read before the rule writes it: afterwards the map no longer says whether
	// this click took the tile.
	before, _ := u.owners.Owner(in.TileID)

	out, err := u.implementation.Execute(ctx, in)
	if err != nil || before == in.CountryID {
		return out, err
	}

	for _, found := range findPockets(in.TileID, in.CountryID, maxTiles, u.owners, u.neighbours) {
		left, ok := u.enclosures.Spend(scope)
		if !ok {
			break
		}

		for _, tile := range found.filled {
			if err := u.storage.Set(ctx, tile, in.CountryID); err != nil {
				return out, fmt.Errorf("failed to take enclosed tile %d: %w", tile, err)
			}
		}

		// After the tiles, so a client never shows a shape filling that the map
		// has not taken yet.
		u.publisher.PublishEnclosed(scope, bonus.Enclosed{
			CountryID:   in.CountryID,
			ClosingTile: in.TileID,
			Wall:        found.wall,
			Filled:      found.filled,
			Left:        left,
		})
	}

	return out, nil
}
