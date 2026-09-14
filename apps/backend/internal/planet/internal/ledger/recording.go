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
	SetBoosted(ctx context.Context, tile uint32, value string) error
}

func NewRecording(tiles Tiles, takings Storage, clock cptime.Clock) Recording {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return Recording{tiles: tiles, takings: takings, clock: clock}
}

// Recording is the tile writer the click chain is handed, so a click, a spread and an enclose all
// land in the ledger under the caller who made them.
type Recording struct {
	tiles   Tiles
	takings Storage
	clock   cptime.Clock
}

func (r Recording) Set(ctx context.Context, tile uint32, value string) error {
	return r.write(ctx, tile, value, r.tiles.Set)
}

func (r Recording) SetBoosted(ctx context.Context, tile uint32, value string) error {
	return r.write(ctx, tile, value, r.tiles.SetBoosted)
}

// The owner and the last take are read apart from the writes, so a click racing this one can leave a stale Previous.
// A revert only restores a tile still wearing this caller's paint, so the worst it does is pick that stale owner.
func (r Recording) write(
	ctx context.Context,
	tile uint32,
	value string,
	set func(context.Context, uint32, string) error,
) error {
	previous, _ := r.tiles.Owner(tile)

	if err := set(ctx, tile, value); err != nil {
		return err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	scope := cpipscope.Of(cpctx.GetSourceIP(ctx))
	if previous == value || scope == "" {
		return nil
	}

	taking := Taking{Tile: tile, Scope: scope, Country: value, Previous: previous, At: r.clock.Now()}
	last, found := r.takings.Last(tile)
	r.takings.Put(taking.Over(last, found))

	return nil
}
