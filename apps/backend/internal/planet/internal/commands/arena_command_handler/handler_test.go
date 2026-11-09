package arena_command_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/close_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/open_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/arena_command_handler"
)

var ends = time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC)

type opens struct {
	told []open_arena_usecase.Command
	err  error
}

func (o *opens) Execute(_ context.Context, command open_arena_usecase.Command) (bool, error) {
	o.told = append(o.told, command)
	return o.err == nil, o.err
}

type closes struct {
	told []arenas.ID
	err  error
}

func (c *closes) Execute(_ context.Context, arena arenas.ID) (close_arena_usecase.Closed, error) {
	c.told = append(c.told, arena)
	return close_arena_usecase.Restored, c.err
}

func openArena(id string, endsAt *timestamppb.Timestamp) *planetv1.ArenaCommand {
	return &planetv1.ArenaCommand{Command: &planetv1.ArenaCommand_OpenArena{OpenArena: &planetv1.OpenArena{ArenaId: id, EndsAt: endsAt}}}
}

func closeArena(id string) *planetv1.ArenaCommand {
	return &planetv1.ArenaCommand{Command: &planetv1.ArenaCommand_CloseArena{CloseArena: &planetv1.CloseArena{ArenaId: id}}}
}

func TestEachCommandGoesToItsUseCase(t *testing.T) {
	open, closing := &opens{}, &closes{}
	handler := arena_command_handler.New(open, closing)

	require.NoError(t, handler.Handle(t.Context(), openArena("battle", timestamppb.New(ends))))
	require.NoError(t, handler.Handle(t.Context(), closeArena("battle")))

	assert.Equal(t, []open_arena_usecase.Command{{Arena: "battle", EndsAt: ends}}, open.told)
	assert.Equal(t, []arenas.ID{"battle"}, closing.told)
}

func TestACommandThatCanNeverApplyIsDropped(t *testing.T) {
	open, closing := &opens{err: fmt.Errorf("%w: closed already", arenas.ErrRefused)}, &closes{}
	handler := arena_command_handler.New(open, closing)

	for name, command := range map[string]*planetv1.ArenaCommand{
		"no arena":    openArena("", timestamppb.New(ends)),
		"no end":      openArena("battle", nil),
		"no close id": closeArena(""),
		"empty":       {},
		"refused":     openArena("battle", timestamppb.New(ends)),
	} {
		require.NoError(t, handler.Handle(t.Context(), command), name)
	}
	assert.Len(t, open.told, 1, "only the refused one reached its use case")
	assert.Empty(t, closing.told)
}

func TestAFailureIsDeliveredAgain(t *testing.T) {
	down := errors.New("postgres is down")
	handler := arena_command_handler.New(&opens{err: down}, &closes{err: down})

	require.ErrorIs(t, handler.Handle(t.Context(), openArena("battle", timestamppb.New(ends))), down)
	require.ErrorIs(t, handler.Handle(t.Context(), closeArena("battle")), down)
}
