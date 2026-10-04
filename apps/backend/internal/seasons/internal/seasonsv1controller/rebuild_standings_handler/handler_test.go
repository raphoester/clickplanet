package rebuild_standings_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/rebuild_standings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/rebuild_standings_usecase"
)

type stubUseCase struct {
	out rebuild_standings_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context) (rebuild_standings_usecase.Out, error) {
	return s.out, s.err
}

func rebuild(t *testing.T, useCase stubUseCase) (*connect.Response[seasonsv1.RebuildStandingsResponse], error) {
	t.Helper()

	return rebuild_standings_handler.New(useCase).RebuildStandings(t.Context(), //nolint:wrapcheck // the tests read the error.
		connect.NewRequest(&seasonsv1.RebuildStandingsRequest{}))
}

func TestTheAnswerSaysWhereTheReplayStarts(t *testing.T) {
	res, err := rebuild(t, stubUseCase{out: rebuild_standings_usecase.Out{From: 42}})

	require.NoError(t, err)
	assert.Equal(t, uint64(42), res.Msg.GetFromPosition())
}

func TestStandingsThatNeverBeganAreAFailedPrecondition(t *testing.T) {
	_, err := rebuild(t, stubUseCase{err: fmt.Errorf("wrapped: %w", standings.ErrNotStarted)})

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestAFailureIsAnError(t *testing.T) {
	failed := errors.New("postgres is down")

	_, err := rebuild(t, stubUseCase{err: failed})

	assert.ErrorIs(t, err, failed)
}
