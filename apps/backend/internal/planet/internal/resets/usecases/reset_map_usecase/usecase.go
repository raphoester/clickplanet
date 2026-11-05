package reset_map_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
)

type Tiles interface {
	Wipe(ctx context.Context, keep func(ctx context.Context) (bool, error)) (bool, error)
}

type Executor interface {
	// Execute is true when the map was wiped: a reset kept already wipes nothing.
	Execute(ctx context.Context, reset resets.Reset) (bool, error)
}

func New(tiles Tiles, kept resets.Store) *UseCase {
	return &UseCase{tiles: tiles, kept: kept}
}

// UseCase wipes the map once per reset, keeping the reset in the wipe's transaction.
type UseCase struct {
	tiles Tiles
	kept  resets.Store
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, reset resets.Reset) (bool, error) {
	return u.tiles.Wipe(ctx, func(ctx context.Context) (bool, error) { //nolint:wrapcheck // the storage names it.
		return u.kept.Keep(ctx, reset) //nolint:wrapcheck // the store names it.
	})
}
