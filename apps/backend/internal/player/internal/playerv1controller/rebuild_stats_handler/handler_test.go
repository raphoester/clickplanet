package rebuild_stats_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/rebuild_stats_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/usecases/rebuild_stats_usecase"
)

type stubUseCase struct {
	out rebuild_stats_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context) (rebuild_stats_usecase.Out, error) {
	return s.out, s.err
}

func rebuild(t *testing.T, useCase stubUseCase) (*connect.Response[playerv1.RebuildStatsResponse], error) {
	t.Helper()

	return rebuild_stats_handler.New(useCase).RebuildStats(t.Context(), //nolint:wrapcheck // the tests read the error.
		connect.NewRequest(&playerv1.RebuildStatsRequest{}))
}

func TestTheAnswerSaysWhereTheReplayStarts(t *testing.T) {
	res, err := rebuild(t, stubUseCase{out: rebuild_stats_usecase.Out{From: 42}})

	require.NoError(t, err)
	assert.Equal(t, uint64(42), res.Msg.GetFromPosition())
}

func TestStatsThatNeverBeganAreAFailedPrecondition(t *testing.T) {
	_, err := rebuild(t, stubUseCase{err: fmt.Errorf("wrapped: %w", takes.ErrNotStarted)})

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestAFailureIsAnError(t *testing.T) {
	failed := errors.New("postgres is down")

	_, err := rebuild(t, stubUseCase{err: failed})

	assert.ErrorIs(t, err, failed)
}
