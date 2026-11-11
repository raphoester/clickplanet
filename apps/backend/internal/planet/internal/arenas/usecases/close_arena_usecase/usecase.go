package close_arena_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Tiles interface {
	PutBack(ctx context.Context, arena clicks.MapID, keep func(ctx context.Context) (bool, error)) (bool, error)
}

type Arenas interface {
	Find(ctx context.Context, id arenas.ID) (arenas.Arena, error)
	Close(ctx context.Context, closing arenas.Closing) error
}

type InPlay interface {
	Close(id arenas.ID)
}

// Closed is what a close did.
type Closed uint8

const (
	// Already closed: nothing changed.
	Already Closed = iota + 1
	// Restored: the arena's map is deleted, and the world's plays again.
	Restored
	// Unopened: the arena never opened, and closed with nothing to bring back.
	Unopened
)

type Executor interface {
	Execute(ctx context.Context, arena arenas.ID) (Closed, error)
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

// Execute deletes the arena's map and keeps the arena closed, in one transaction under the map's locks; the world's map
// plays again. An arena never opened is closed the same way, with the map left as it is.
func (u *UseCase) Execute(ctx context.Context, id arenas.ID) (Closed, error) {
	closed := Already
	_, err := u.tiles.PutBack(ctx, id.Map(), func(ctx context.Context) (bool, error) {
		arena, err := u.kept.Find(ctx, id)
		if err != nil {
			return false, err //nolint:wrapcheck // the store names it.
		}
		closing, closes := arena.Close(u.clock.Now())
		if !closes {
			return false, nil
		}
		if err := u.kept.Close(ctx, closing); err != nil {
			return false, err //nolint:wrapcheck // the store names it.
		}
		closed = Unopened
		if arena.Opened {
			closed = Restored
		}
		return arena.Opened, nil
	})
	if err != nil {
		return 0, err //nolint:wrapcheck // the storage names it.
	}
	if closed != Already {
		u.inPlay.Close(id)
	}
	return closed, nil
}
