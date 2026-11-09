package close_arena_usecase

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Tiles interface {
	PutBack(ctx context.Context, take func(ctx context.Context) (clicks.MapCopy, bool, error)) (bool, error)
}

type Arenas interface {
	Find(ctx context.Context, id arenas.ID) (arenas.Arena, error)
	Close(ctx context.Context, closing arenas.Closing) (clicks.MapCopy, error)
}

type InPlay interface {
	Close(id arenas.ID)
}

// Closed is what a close did.
type Closed uint8

const (
	// Already closed: nothing changed.
	Already Closed = iota + 1
	// Restored: the map from before the arena is back.
	Restored
	// Unopened: the arena never opened, and closed with nothing to bring back.
	Unopened
)

type Executor interface {
	Execute(ctx context.Context, arena arenas.ID) (Closed, error)
}

func New(tiles Tiles, kept Arenas, outbox cpatomic.Outbox, inPlay InPlay, clock cptime.Clock) *UseCase {
	return &UseCase{tiles: tiles, kept: kept, outbox: outbox, inPlay: inPlay, clock: clock}
}

type UseCase struct {
	tiles  Tiles
	kept   Arenas
	outbox cpatomic.Outbox
	inPlay InPlay
	clock  cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// Execute brings the map from before the arena back, keeps it closed and records ArenaClosed, in one transaction under
// the map's locks. An arena never opened is closed and told the same way, with the map left as it is.
func (u *UseCase) Execute(ctx context.Context, id arenas.ID) (Closed, error) {
	closed := Already
	_, err := u.tiles.PutBack(ctx, func(ctx context.Context) (clicks.MapCopy, bool, error) {
		arena, err := u.kept.Find(ctx, id)
		if err != nil {
			return clicks.MapCopy{}, false, err //nolint:wrapcheck // the store names it.
		}
		closing, closes := arena.Close(u.clock.Now())
		if !closes {
			return clicks.MapCopy{}, false, nil
		}
		before, err := u.kept.Close(ctx, closing)
		if err != nil {
			return clicks.MapCopy{}, false, err //nolint:wrapcheck // the store names it.
		}
		if err := u.outbox.Record(ctx, &planetv1.ArenaClosed{
			ArenaId: string(id), ClosedAt: timestamppb.New(closing.ClosedAt),
		}); err != nil {
			return clicks.MapCopy{}, false, err //nolint:wrapcheck // the outbox names it.
		}
		closed = Unopened
		if arena.Opened {
			closed = Restored
		}
		return before, arena.Opened, nil
	})
	if err != nil {
		return 0, err //nolint:wrapcheck // the storage names it.
	}
	if closed != Already {
		u.inPlay.Close(id)
	}
	return closed, nil
}
