package unban_player_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/unban_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/unban_player_handler"
)

type stubUseCase struct {
	in  unban_player_usecase.In
	out unban_player_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in unban_player_usecase.In) (unban_player_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func unban(t *testing.T, useCase *stubUseCase, req *planetv1.UnbanPlayerRequest) (*planetv1.UnbanPlayerResponse, error) {
	t.Helper()

	res, err := unban_player_handler.New(useCase).UnbanPlayer(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("unban refused: %w", err)
	}

	return res.Msg, nil
}

func TestTheRequestReachesTheUseCaseAndTheLiftedBanComesBack(t *testing.T) {
	until := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{out: unban_player_usecase.Out{Scope: "9.9.9.9", Offence: 2, Until: until}}

	res, err := unban(t, useCase, &planetv1.UnbanPlayerRequest{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Equal(t, unban_player_usecase.In{Scope: "9.9.9.9"}, useCase.in)
	assert.Equal(t, "9.9.9.9", res.GetScope())
	assert.Empty(t, res.GetAccountId())
	assert.Equal(t, uint32(2), res.GetOffence())
	assert.Equal(t, until, res.GetBannedUntil().AsTime())
}

func TestAnAccountReachesTheUseCaseAndComesBack(t *testing.T) {
	const account = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	useCase := &stubUseCase{out: unban_player_usecase.Out{Account: account, Offence: 1}}

	res, err := unban(t, useCase, &planetv1.UnbanPlayerRequest{AccountId: account})
	require.NoError(t, err)

	assert.Equal(t, unban_player_usecase.In{Account: account}, useCase.in)
	assert.Equal(t, account, res.GetAccountId())
	assert.Empty(t, res.GetScope())
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	cause := errors.New("boom")

	for err, code := range map[error]connect.Code{
		fmt.Errorf("%w: %q", ledger.ErrInvalidScope, "bot"):     connect.CodeInvalidArgument,
		fmt.Errorf("%w: %q", ledger.ErrInvalidAccount, "guest"): connect.CodeInvalidArgument,
		ledger.ErrNoCaller:                 connect.CodeInvalidArgument,
		unban_player_usecase.ErrNotBanned:  connect.CodeNotFound,
		unban_player_usecase.ErrAntiBotOff: connect.CodeFailedPrecondition,
		cause:                              connect.CodeUnknown,
	} {
		_, got := unban(t, &stubUseCase{err: err}, &planetv1.UnbanPlayerRequest{Scope: "bot"})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
