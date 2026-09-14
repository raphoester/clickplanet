package paint_random_tiles_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/paint_random_tiles_handler"
)

type stubUseCase struct {
	in  paint_random_tiles_usecase.In
	out paint_random_tiles_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in paint_random_tiles_usecase.In) (paint_random_tiles_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func paint(t *testing.T, useCase *stubUseCase) (*planetv1.PaintRandomTilesResponse, error) {
	t.Helper()

	res, err := paint_random_tiles_handler.New(useCase).PaintRandomTiles(t.Context(),
		connect.NewRequest(&planetv1.PaintRandomTilesRequest{
			FlagCountryId: "dz", AreaCountryId: "fr", Count: 500, Proximity: 0.8, DryRun: true,
		}))
	if err != nil {
		return nil, fmt.Errorf("paint refused: %w", err)
	}

	return res.Msg, nil
}

func TestTheRequestReachesTheUseCaseAndTheCountsComeBack(t *testing.T) {
	useCase := &stubUseCase{out: paint_random_tiles_usecase.Out{Eligible: 9000, Picked: 500, Painted: 498}}

	res, err := paint(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 500, Proximity: 0.8, DryRun: true}, useCase.in)
	assert.Equal(t, uint32(9000), res.GetEligible())
	assert.Equal(t, uint32(500), res.GetPicked())
	assert.Equal(t, uint32(498), res.GetPainted())
}

func TestACallerMistakeIsInvalidArgument(t *testing.T) {
	for _, err := range []error{
		fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, "xx"),
		fmt.Errorf("%w: %d", paint_random_tiles_usecase.ErrInvalidCount, 0),
		fmt.Errorf("%w: %v", paint_random_tiles_usecase.ErrInvalidProximity, 2),
	} {
		_, got := paint(t, &stubUseCase{err: err})
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(got), err.Error())
	}
}

func TestAnythingElseIsLeftToTheErrorNet(t *testing.T) {
	cause := errors.New("paint interrupted: context canceled")

	_, err := paint(t, &stubUseCase{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err), "an unrecognised error must reach the net unwrapped")
}
