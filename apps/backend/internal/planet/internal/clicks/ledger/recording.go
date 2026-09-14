package ledger

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

type Tiles interface {
	Owner(tile uint32) (string, bool)
	Set(ctx context.Context, tile uint32, value string) error
	SetBoosted(ctx context.Context, tile uint32, value string) error
}

// Recording is the tile writer the click chain is handed, so a click, a spread and an enclose all
// land in the ledger under the caller who made them.
type Recording struct {
	Tiles  Tiles
	Ledger *Ledger
}

func (r Recording) Set(ctx context.Context, tile uint32, value string) error {
	return r.write(ctx, tile, value, r.Tiles.Set)
}

func (r Recording) SetBoosted(ctx context.Context, tile uint32, value string) error {
	return r.write(ctx, tile, value, r.Tiles.SetBoosted)
}

// The owner is read apart from the write, so a click racing this one can leave a stale Previous.
// A revert only restores a tile still wearing this caller's paint, so the worst it does is pick that stale owner.
func (r Recording) write(
	ctx context.Context,
	tile uint32,
	value string,
	set func(context.Context, uint32, string) error,
) error {
	previous, _ := r.Tiles.Owner(tile)

	if err := set(ctx, tile, value); err != nil {
		return err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	if previous != value {
		r.Ledger.Record(tile, cpipscope.Of(cpctx.GetSourceIP(ctx)), previous, value)
	}

	return nil
}
