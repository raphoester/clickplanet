package close_arena_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Tiles interface {
	PutBack(ctx context.Context, arena clicks.MapID, world clicks.Stage, keep func(ctx context.Context) (bool, error)) (bool, error)
}

// Maps names the stage a map plays: its latest era's board, when this process carries it.
type Maps interface {
	Playing(ctx context.Context, played clicks.MapID) (clicks.Stage, error)
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

func New(tiles Tiles, maps Maps, kept Arenas, inPlay InPlay, clock cptime.Clock) *UseCase {
	return &UseCase{tiles: tiles, maps: maps, kept: kept, inPlay: inPlay, clock: clock}
}

type UseCase struct {
	tiles  Tiles
	maps   Maps
	kept   Arenas
	inPlay InPlay
	clock  cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// Execute deletes the arena's map and keeps the arena closed, in one transaction under the map's locks; the world's map
// plays again, on the board of its era, whatever board the arena played. An arena never opened is closed the same way,
// with the map left as it is.
func (u *UseCase) Execute(ctx context.Context, id arenas.ID) (Closed, error) {
	world, err := u.maps.Playing(ctx, clicks.World)
	if err != nil {
		return 0, fmt.Errorf("failed to read the board the world plays: %w", err)
	}
	closed := Already
	_, err = u.tiles.PutBack(ctx, id.Map(), world, func(ctx context.Context) (bool, error) {
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
