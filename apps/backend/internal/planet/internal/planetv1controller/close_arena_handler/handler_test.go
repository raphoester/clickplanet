package close_arena_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/close_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/close_arena_handler"
)

// closing stops on a cancelled context, as a swap of the map in postgres does.
type closing struct {
	asked  []arenas.ID
	closed close_arena_usecase.Closed
	err    error
}

func (c *closing) Execute(ctx context.Context, arena arenas.ID) (close_arena_usecase.Closed, error) {
	if err := ctx.Err(); err != nil {
		return 0, err //nolint:wrapcheck // the context names it.
	}
	c.asked = append(c.asked, arena)
	return c.closed, c.err
}

func closeArena(ctx context.Context, useCase *closing, id string) error {
	_, err := close_arena_handler.New(useCase).CloseArena(ctx, connect.NewRequest(&planetv1.CloseArenaRequest{ArenaId: id}))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func TestAnArenaClosedNowBeforeOrNeverOpenedAnswersTheSame(t *testing.T) {
	for _, closed := range []close_arena_usecase.Closed{
		close_arena_usecase.Restored, close_arena_usecase.Already, close_arena_usecase.Unopened,
	} {
		useCase := &closing{closed: closed}

		require.NoError(t, closeArena(t.Context(), useCase, "battle"))

		assert.Equal(t, []arenas.ID{"battle"}, useCase.asked)
	}
}

func TestABadIDIsAnInvalidArgument(t *testing.T) {
	useCase := &closing{closed: close_arena_usecase.Restored}

	for _, id := range []string{"", "world"} {
		err := closeArena(t.Context(), useCase, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
		require.ErrorIs(t, err, arenas.ErrInvalidArena, id)
	}
	assert.Empty(t, useCase.asked)
}

func TestTheSwapGoesOnWhenTheCallerGivesUp(t *testing.T) {
	useCase := &closing{closed: close_arena_usecase.Restored}
	gaveUp, cancel := context.WithCancel(t.Context())
	cancel()

	require.NoError(t, closeArena(gaveUp, useCase, "battle"))

	assert.Equal(t, []arenas.ID{"battle"}, useCase.asked)
}
