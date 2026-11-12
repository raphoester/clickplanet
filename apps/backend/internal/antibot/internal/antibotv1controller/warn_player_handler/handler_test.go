package warn_player_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/warn_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/warn_player_usecase"
)

const account = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

type stubUseCase struct {
	in  warn_player_usecase.In
	out warn_player_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in warn_player_usecase.In) (warn_player_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func warn(t *testing.T, useCase *stubUseCase, req *antibotv1.WarnPlayerRequest) (*antibotv1.WarnPlayerResponse, error) {
	t.Helper()

	res, err := warn_player_handler.New(useCase).WarnPlayer(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("warning refused: %w", err)
	}

	return res.Msg, nil
}

func TestTheAccountReachesTheUseCaseAndItsWarningComesBack(t *testing.T) {
	at := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{out: warn_player_usecase.Out{Account: account, WarnedAt: at}}

	res, err := warn(t, useCase, &antibotv1.WarnPlayerRequest{AccountId: account})
	require.NoError(t, err)

	assert.Equal(t, warn_player_usecase.In{Account: account}, useCase.in)
	assert.Equal(t, account, res.GetAccountId())
	assert.Equal(t, at, res.GetWarnedAt().AsTime())
	assert.False(t, res.GetAlreadyWarned())
	assert.Nil(t, res.GetAcknowledgedAt(), "not acknowledged yet")
}

func TestAWarningGivenBeforeComesBackWithItsAcknowledgement(t *testing.T) {
	at := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	acknowledged := at.Add(time.Minute)
	useCase := &stubUseCase{out: warn_player_usecase.Out{
		Account: account, WarnedAt: at, AlreadyWarned: true, AcknowledgedAt: acknowledged,
	}}

	res, err := warn(t, useCase, &antibotv1.WarnPlayerRequest{AccountId: account})
	require.NoError(t, err)

	assert.True(t, res.GetAlreadyWarned())
	assert.Equal(t, at, res.GetWarnedAt().AsTime())
	assert.Equal(t, acknowledged, res.GetAcknowledgedAt().AsTime())
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	cause := errors.New("boom")

	for err, code := range map[error]connect.Code{
		fmt.Errorf("%w: %q", bans.ErrInvalidAccount, "not-an-account"): connect.CodeInvalidArgument,
		warn_player_usecase.ErrNotEnforced:                             connect.CodeFailedPrecondition,
		cause:                                                          connect.CodeUnknown,
	} {
		_, got := warn(t, &stubUseCase{err: err}, &antibotv1.WarnPlayerRequest{AccountId: "not-an-account"})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
