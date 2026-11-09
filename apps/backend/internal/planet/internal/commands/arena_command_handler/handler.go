package arena_command_handler

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/close_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/open_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

func New(open open_arena_usecase.Executor, closing close_arena_usecase.Executor) Handler {
	return Handler{open: open, closing: closing}
}

type Handler struct {
	open    open_arena_usecase.Executor
	closing close_arena_usecase.Executor
}

var _ cpredis.Handler[*planetv1.ArenaCommand] = Handler{}

// A command that can never apply is dropped: delivered again, it would hold back the commands behind it on the one
// stream. No timeout of its own: an open or a close writes the whole map in one transaction, and a cut one is delivered
// again.
func (h Handler) Handle(ctx context.Context, command *planetv1.ArenaCommand) error {
	switch kind := command.GetCommand().(type) {
	case *planetv1.ArenaCommand_OpenArena:
		return h.openArena(ctx, kind.OpenArena)
	case *planetv1.ArenaCommand_CloseArena:
		return h.closeArena(ctx, kind.CloseArena)
	case nil:
		return nil
	}
	return fmt.Errorf("an arena command of a kind this build does not know: %T", command.GetCommand())
}

func (h Handler) openArena(ctx context.Context, command *planetv1.OpenArena) error {
	id, err := arenas.ParseID(command.GetArenaId())
	if err != nil || command.GetEndsAt() == nil {
		return nil
	}
	_, err = h.open.Execute(ctx, open_arena_usecase.Command{Arena: id, EndsAt: command.GetEndsAt().AsTime()})
	if errors.Is(err, arenas.ErrRefused) {
		return nil
	}
	return err //nolint:wrapcheck // the use case named it.
}

func (h Handler) closeArena(ctx context.Context, command *planetv1.CloseArena) error {
	id, err := arenas.ParseID(command.GetArenaId())
	if err != nil {
		return nil
	}
	_, err = h.closing.Execute(ctx, id)
	return err //nolint:wrapcheck // the use case named it.
}
