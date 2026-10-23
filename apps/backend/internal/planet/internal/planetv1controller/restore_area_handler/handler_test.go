package restore_area_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/restore_area_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/restore_area_handler"
)

type stubUseCase struct {
	in  restore_area_usecase.In
	out restore_area_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in restore_area_usecase.In) (restore_area_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

var before = time.Date(2026, 10, 20, 11, 0, 0, 0, time.UTC)

func restore(t *testing.T, useCase *stubUseCase, req *planetv1.RestoreAreaRequest) (*planetv1.RestoreAreaResponse, error) {
	t.Helper()

	res, err := restore_area_handler.New(useCase).RestoreArea(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("restore refused: %w", err)
	}
	return res.Msg, nil
}

func TestTheRequestReachesTheUseCaseAndTheCountsComeBack(t *testing.T) {
	useCase := &stubUseCase{out: restore_area_usecase.Out{
		Tiles: 37, Changed: 12, Restored: 11, PaintersTotal: 3, KeptSince: before.Add(-71 * time.Hour),
		Painters: []ledger.Player{{Scope: "9.9.9.9", Account: "account-1", Flag: "de", Takes: 12}},
	}}

	res, err := restore(t, useCase, &planetv1.RestoreAreaRequest{TileId: 42, Rings: 3, Before: timestamppb.New(before)})
	require.NoError(t, err)

	assert.Equal(t, restore_area_usecase.In{Tile: 42, Rings: 3, Before: before}, useCase.in)
	assert.Equal(t, uint32(37), res.GetTiles())
	assert.Equal(t, uint32(12), res.GetChanged())
	assert.Equal(t, uint32(11), res.GetRestored())
	assert.Equal(t, uint32(3), res.GetPaintersTotal())
	require.Len(t, res.GetPainters(), 1)
	assert.Equal(t, "account-1", res.GetPainters()[0].GetAccountId())
	assert.Equal(t, before.Add(-71*time.Hour), res.GetKeptSince().AsTime())
}

func TestNoTimeReachesTheUseCaseAsNone(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := restore(t, useCase, &planetv1.RestoreAreaRequest{TileId: 42, Rings: 3})
	require.NoError(t, err)

	assert.True(t, useCase.in.Before.IsZero())
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("refused: %w", clicks.ErrTileOutOfRange):              connect.CodeInvalidArgument,
		fmt.Errorf("refused: %w", clicks.ErrInvalidArea):                 connect.CodeInvalidArgument,
		restore_area_usecase.ErrNoTime:                                   connect.CodeInvalidArgument,
		fmt.Errorf("refused: %w", restore_area_usecase.ErrPastTheLedger): connect.CodeFailedPrecondition,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := restore(t, &stubUseCase{err: err}, &planetv1.RestoreAreaRequest{TileId: 42, Rings: 3})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
