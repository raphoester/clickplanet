package open_arena_handler_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/open_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/open_arena_handler"
)

var ends = time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC)

// opening stops on a cancelled context, as a swap of the map in postgres does.
type opening struct {
	asked   []open_arena_usecase.Command
	already bool
	err     error
}

func (o *opening) Execute(ctx context.Context, command open_arena_usecase.Command) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err //nolint:wrapcheck // the context names it.
	}
	o.asked = append(o.asked, command)
	return !o.already && o.err == nil, o.err
}

func open(ctx context.Context, useCase *opening, req *planetv1.OpenArenaRequest) error {
	_, err := open_arena_handler.New(useCase).OpenArena(ctx, connect.NewRequest(req))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func battle() *planetv1.OpenArenaRequest {
	return &planetv1.OpenArenaRequest{ArenaId: "battle", EndsAt: timestamppb.New(ends), RuleSet: "siege"}
}

func TestTheArenaOpensForItsBattleWithItsSet(t *testing.T) {
	useCase := &opening{}

	require.NoError(t, open(t.Context(), useCase, battle()))

	assert.Equal(t, []open_arena_usecase.Command{{Arena: "battle", EndsAt: ends, Set: "siege"}}, useCase.asked)
}

func TestAnArenaOpenAlreadyAnswersTheSame(t *testing.T) {
	require.NoError(t, open(t.Context(), &opening{already: true}, battle()))
}

func TestABadIDOrNoEndIsAnInvalidArgument(t *testing.T) {
	useCase := &opening{}

	for name, req := range map[string]*planetv1.OpenArenaRequest{
		"no id":    {EndsAt: timestamppb.New(ends), RuleSet: "siege"},
		"world's":  {ArenaId: "world", EndsAt: timestamppb.New(ends), RuleSet: "siege"},
		"no end":   {ArenaId: "battle", RuleSet: "siege"},
		"too long": {ArenaId: strings.Repeat("a", 65), EndsAt: timestamppb.New(ends), RuleSet: "siege"},
	} {
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(open(t.Context(), useCase, req)), name)
	}
	assert.Empty(t, useCase.asked)
}

func TestARefusalIsAFailedPrecondition(t *testing.T) {
	err := open(t.Context(), &opening{err: fmt.Errorf("%w: arena battle closed already", arenas.ErrRefused)}, battle())

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorIs(t, err, arenas.ErrRefused)
}

func TestTheSwapGoesOnWhenTheCallerGivesUp(t *testing.T) {
	useCase := &opening{}
	gaveUp, cancel := context.WithCancel(t.Context())
	cancel()

	require.NoError(t, open(gaveUp, useCase, battle()))

	assert.Len(t, useCase.asked, 1)
}
