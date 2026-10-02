package ledger

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Tiles interface {
	Owner(tile uint32) (string, bool)
	Set(ctx context.Context, tile uint32, value string) error
}

func NewRecording(tiles Tiles, takings Storage, clock cptime.Clock) Recording {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return Recording{tiles: tiles, takings: takings, clock: clock}
}

type Recording struct {
	tiles   Tiles
	takings Storage
	clock   cptime.Clock
}

func (r Recording) Owner(tile uint32) (string, bool) {
	return r.tiles.Owner(tile)
}

func (r Recording) Set(ctx context.Context, tile uint32, value string) error {
	previous, _ := r.tiles.Owner(tile)

	if err := r.tiles.Set(ctx, tile, value); err != nil {
		return err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	scope := cpipscope.Of(cpctx.GetSourceIP(ctx))
	if previous == value || scope == "" {
		return nil
	}

	r.takings.Append(Taking{
		Tile: tile, Scope: scope, Account: cpctx.GetAccount(ctx), Country: value, Previous: previous, At: r.clock.Now(),
	})

	return nil
}
