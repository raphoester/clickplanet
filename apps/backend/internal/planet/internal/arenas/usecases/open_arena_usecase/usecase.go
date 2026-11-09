package open_arena_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Tiles interface {
	Blank(ctx context.Context, keep func(ctx context.Context, before clicks.MapCopy) (bool, error)) (bool, error)
}

type Arenas interface {
	Find(ctx context.Context, id arenas.ID) (arenas.Arena, error)
	Playing(ctx context.Context) (arenas.Arena, error)
	Open(ctx context.Context, opening arenas.Opening, before clicks.MapCopy) error
}

type InPlay interface {
	Open(opening arenas.Opening)
}

type Command struct {
	Arena  arenas.ID
	EndsAt time.Time
}

type Executor interface {
	// Execute answers whether it opened the arena: one open already is false.
	Execute(ctx context.Context, command Command) (bool, error)
}

func New(tiles Tiles, kept Arenas, inPlay InPlay, clock cptime.Clock) *UseCase {
	return &UseCase{tiles: tiles, kept: kept, inPlay: inPlay, clock: clock}
}

type UseCase struct {
	tiles  Tiles
	kept   Arenas
	inPlay InPlay
	clock  cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// Execute keeps the map from before the arena and makes it blank, in one transaction, under the map's locks: the
// arena is read there, so a command told twice opens it once.
func (u *UseCase) Execute(ctx context.Context, command Command) (bool, error) {
	var opening arenas.Opening
	opened, err := u.tiles.Blank(ctx, func(ctx context.Context, before clicks.MapCopy) (bool, error) {
		arena, err := u.kept.Find(ctx, command.Arena)
		if err != nil {
			return false, err //nolint:wrapcheck // the store names it.
		}
		playing, err := u.kept.Playing(ctx)
		if err != nil {
			return false, err //nolint:wrapcheck // the store names it.
		}
		next, opens, err := arena.Open(command.EndsAt, u.clock.Now(), playing)
		if err != nil || !opens {
			return false, err //nolint:wrapcheck // the arena says why.
		}
		if err := u.kept.Open(ctx, next, before); err != nil {
			return false, err //nolint:wrapcheck // the store names it.
		}
		opening = next
		return true, nil
	})
	if err != nil || !opened {
		return false, err //nolint:wrapcheck // the storage names it.
	}
	u.inPlay.Open(opening)
	return true, nil
}
