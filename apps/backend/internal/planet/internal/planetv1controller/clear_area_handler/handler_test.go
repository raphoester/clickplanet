package clear_area_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/clear_area_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/clear_area_handler"
)

type stubUseCase struct {
	in  clear_area_usecase.In
	out clear_area_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in clear_area_usecase.In) (clear_area_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func clearArea(t *testing.T, useCase *stubUseCase) (*planetv1.ClearAreaResponse, error) {
	t.Helper()

	res, err := clear_area_handler.New(useCase).ClearArea(t.Context(),
		connect.NewRequest(&planetv1.ClearAreaRequest{TileId: 42, Rings: 3, DryRun: true}))
	if err != nil {
		return nil, fmt.Errorf("clear refused: %w", err)
	}
	return res.Msg, nil
}

func TestTheRequestReachesTheUseCaseAndTheCountsComeBack(t *testing.T) {
	useCase := &stubUseCase{out: clear_area_usecase.Out{Tiles: 37, Owned: 20}}

	res, err := clearArea(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, clear_area_usecase.In{Tile: 42, Rings: 3, DryRun: true}, useCase.in)
	assert.Equal(t, uint32(37), res.GetTiles())
	assert.Equal(t, uint32(20), res.GetOwned())
	assert.Zero(t, res.GetCleared())
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("refused: %w", clicks.ErrTileOutOfRange): connect.CodeInvalidArgument,
		fmt.Errorf("refused: %w", clicks.ErrInvalidArea):    connect.CodeInvalidArgument,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := clearArea(t, &stubUseCase{err: err})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
