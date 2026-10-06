package ledger

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Tiles interface {
	Owner(tile uint32) (string, bool)
	Set(ctx context.Context, tile uint32, value string) error
	Click(ctx context.Context, tile uint32, value string) error
	Clear(ctx context.Context, blast clicks.Blast) (clicks.Blast, error)
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
	return r.record(ctx, tile, value, r.tiles.Set)
}

func (r Recording) Click(ctx context.Context, tile uint32, value string) error {
	return r.record(ctx, tile, value, r.tiles.Click)
}

func (r Recording) Clear(ctx context.Context, blast clicks.Blast) (clicks.Blast, error) {
	cleared, err := r.tiles.Clear(ctx, blast)
	if err != nil {
		return cleared, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	if scope := cpipscope.Of(cpctx.GetSourceIP(ctx)); scope != "" {
		r.takings.AppendBombing(Bombing{Scope: scope, Account: cpctx.GetAccount(ctx), At: r.clock.Now(), Blast: cleared})
	}

	return cleared, nil
}

func (r Recording) record(ctx context.Context, tile uint32, value string, write func(context.Context, uint32, string) error) error {
	previous, _ := r.tiles.Owner(tile)

	if err := write(ctx, tile, value); err != nil {
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
