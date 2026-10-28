package add_season_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/add_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type stubUseCase struct {
	err   error
	asked []season.Draft
}

func (s *stubUseCase) Execute(_ context.Context, draft season.Draft) (season.Number, error) {
	s.asked = append(s.asked, draft)
	return 1, s.err
}

func add(t *testing.T, useCase *stubUseCase, req *gamemasterv1.AddSeasonRequest) (*connect.Response[gamemasterv1.AddSeasonResponse], error) {
	t.Helper()
	return add_season_handler.New(useCase).AddSeason(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestTheSeasonAskedIsAddedAndNumbered(t *testing.T) {
	useCase := &stubUseCase{}

	res, err := add(t, useCase, &gamemasterv1.AddSeasonRequest{
		BreakBefore: durationpb.New(time.Hour), Length: durationpb.New(28 * 24 * time.Hour),
	})

	require.NoError(t, err)
	assert.Equal(t, uint32(1), res.Msg.GetNumber())
	assert.Equal(t, []season.Draft{{Break: time.Hour, Length: 28 * 24 * time.Hour}}, useCase.asked)
}

func TestAnUnsetDurationIsNone(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := add(t, useCase, &gamemasterv1.AddSeasonRequest{})

	require.NoError(t, err)
	assert.Equal(t, []season.Draft{{}}, useCase.asked)
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for err, code := range map[error]connect.Code{
		season.ErrInvalidSeason: connect.CodeInvalidArgument,
		season.ErrUnknownSeason: connect.CodeNotFound,
		season.ErrStarted:       connect.CodeFailedPrecondition,
		season.ErrEnded:         connect.CodeFailedPrecondition,
		season.ErrNotLast:       connect.CodeFailedPrecondition,
	} {
		_, got := add(t, &stubUseCase{err: err}, &gamemasterv1.AddSeasonRequest{})

		assert.Equal(t, code, connect.CodeOf(got), "%v", err)
	}
}

func TestAFailureIsTheErrorNets(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := add(t, &stubUseCase{err: down}, &gamemasterv1.AddSeasonRequest{})

	require.ErrorIs(t, err, down)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
