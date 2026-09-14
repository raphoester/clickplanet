package revert_player_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/revert_player_handler"
)

type stubUseCase struct {
	in  revert_player_usecase.In
	out revert_player_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in revert_player_usecase.In) (revert_player_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func revert(t *testing.T, useCase *stubUseCase) (*planetv1.RevertPlayerResponse, error) {
	t.Helper()

	res, err := revert_player_handler.New(useCase).RevertPlayer(t.Context(),
		connect.NewRequest(&planetv1.RevertPlayerRequest{Scope: "9.9.9.9", DryRun: true}))
	if err != nil {
		return nil, fmt.Errorf("revert refused: %w", err)
	}

	return res.Msg, nil
}

func TestTheRequestReachesTheUseCaseAndTheCountsComeBack(t *testing.T) {
	useCase := &stubUseCase{out: revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 40, Held: 31, Restored: 30}}

	res, err := revert(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.In{Scope: "9.9.9.9", DryRun: true}, useCase.in)
	assert.Equal(t, "9.9.9.9", res.GetScope())
	assert.Equal(t, uint32(40), res.GetTouched())
	assert.Equal(t, uint32(31), res.GetHeld())
	assert.Equal(t, uint32(30), res.GetRestored())
}

func TestAnInvalidScopeIsInvalidArgument(t *testing.T) {
	_, err := revert(t, &stubUseCase{err: fmt.Errorf("%w: %q", clicks.ErrInvalidScope, "bot")})
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAnythingElseIsLeftToTheErrorNet(t *testing.T) {
	cause := errors.New("revert interrupted: context canceled")

	_, err := revert(t, &stubUseCase{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
