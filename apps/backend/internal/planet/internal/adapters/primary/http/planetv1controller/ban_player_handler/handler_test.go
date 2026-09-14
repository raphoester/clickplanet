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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/ban_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/ban_player"
)

type stubUseCase struct {
	in  ban_player.In
	out ban_player.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in ban_player.In) (ban_player.Out, error) {
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
	useCase := &stubUseCase{out: ban_player.Out{Scope: "9.9.9.9", Offence: 2, Until: until, Enforced: true}}

	res, err := ban(t, useCase, &planetv1.BanPlayerRequest{Scope: "9.9.9.9", Duration: durationpb.New(time.Hour)})
	require.NoError(t, err)

	assert.Equal(t, ban_player.In{Scope: "9.9.9.9", Duration: time.Hour}, useCase.in)
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
		fmt.Errorf("%w: %q", clicks.ErrInvalidScope, "bot"):   connect.CodeInvalidArgument,
		fmt.Errorf("%w: -1h", ban_player.ErrNegativeDuration): connect.CodeInvalidArgument,
		ban_player.ErrAntiBotOff:                              connect.CodeFailedPrecondition,
		cause:                                                 connect.CodeUnknown,
	} {
		_, got := ban(t, &stubUseCase{err: err}, &planetv1.BanPlayerRequest{Scope: "bot"})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
