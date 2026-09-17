package ban_player_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/ban_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/ban_player_handler"
)

type stubUseCase struct {
	in  ban_player_usecase.In
	out ban_player_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in ban_player_usecase.In) (ban_player_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func ban(t *testing.T, useCase *stubUseCase, req *planetv1.BanPlayerRequest) (*planetv1.BanPlayerResponse, error) {
	t.Helper()

	res, err := ban_player_handler.New(useCase).BanPlayer(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("ban refused: %w", err)
	}

	return res.Msg, nil
}

func TestTheRequestReachesTheUseCaseAndTheSentenceComesBack(t *testing.T) {
	until := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{out: ban_player_usecase.Out{Scope: "9.9.9.9", Offence: 2, Until: until, Enforced: true}}

	res, err := ban(t, useCase, &planetv1.BanPlayerRequest{Scope: "9.9.9.9", Duration: durationpb.New(time.Hour)})
	require.NoError(t, err)

	assert.Equal(t, ban_player_usecase.In{Scope: "9.9.9.9", Duration: time.Hour}, useCase.in)
	assert.Equal(t, "9.9.9.9", res.GetScope())
	assert.Equal(t, uint32(2), res.GetOffence())
	assert.Equal(t, until, res.GetBannedUntil().AsTime())
	assert.True(t, res.GetEnforced())
}

func TestNoDurationTakesTheLadder(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := ban(t, useCase, &planetv1.BanPlayerRequest{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Zero(t, useCase.in.Duration)
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	cause := errors.New("boom")

	for err, code := range map[error]connect.Code{
		fmt.Errorf("%w: %q", ledger.ErrInvalidScope, "bot"):     connect.CodeInvalidArgument,
		fmt.Errorf("%w: %q", ledger.ErrInvalidAccount, "guest"): connect.CodeInvalidArgument,
		ledger.ErrNoCaller: connect.CodeInvalidArgument,
		fmt.Errorf("%w: -1h", ban_player_usecase.ErrNegativeDuration): connect.CodeInvalidArgument,
		ban_player_usecase.ErrAntiBotOff:                              connect.CodeFailedPrecondition,
		cause:                                                         connect.CodeUnknown,
	} {
		_, got := ban(t, &stubUseCase{err: err}, &planetv1.BanPlayerRequest{Scope: "bot"})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}

func TestAnAccountReachesTheUseCaseAndComesBack(t *testing.T) {
	const guest = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	useCase := &stubUseCase{out: ban_player_usecase.Out{Account: guest, Offence: 1}}

	res, err := ban(t, useCase, &planetv1.BanPlayerRequest{AccountId: guest})
	require.NoError(t, err)

	assert.Equal(t, ban_player_usecase.In{Account: guest}, useCase.in)
	assert.Equal(t, guest, res.GetAccountId())
	assert.Empty(t, res.GetScope())
}
