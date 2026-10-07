package mute_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/mute_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
)

type stubUseCase struct {
	in   mute_usecase.In
	mute mutes.Mute
	err  error
}

func (s *stubUseCase) Execute(_ context.Context, in mute_usecase.In) (mutes.Mute, error) {
	s.in = in
	return s.mute, s.err
}

const bullyID = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

func mute(t *testing.T, useCase *stubUseCase, req *chatv1.MuteRequest) (*chatv1.MuteResponse, error) {
	t.Helper()

	res, err := mute_handler.New(useCase).Mute(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("mute refused: %w", err)
	}
	return res.Msg, nil
}

func TestTheRequestReachesTheUseCaseAndTheMuteComesBack(t *testing.T) {
	at := time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	account := messages.AccountIDOf(bullyID)
	useCase := &stubUseCase{mute: mutes.NewMute(mutes.MuteID{1}, mutes.CallerOf(account, "2a01:e0a:1:2::/64"), at, 2*time.Hour)}

	res, err := mute(t, useCase, &chatv1.MuteRequest{AccountId: bullyID, Duration: durationpb.New(2 * time.Hour)})
	require.NoError(t, err)

	assert.Equal(t, mute_usecase.In{Account: account, Duration: 2 * time.Hour}, useCase.in)
	assert.Equal(t, "2a01:e0a:1:2::/64", res.GetScope())
	assert.Equal(t, at.Add(2*time.Hour), res.GetMutedUntil().AsTime())
}

func TestNoDurationIsLeftToTheDefault(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := mute(t, useCase, &chatv1.MuteRequest{AccountId: bullyID})
	require.NoError(t, err)

	assert.Zero(t, useCase.in.Duration)
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		mutes.ErrNoAccount: connect.CodeInvalidArgument,
		fmt.Errorf("cannot mute: %w", mutes.ErrInvalidDuration): connect.CodeInvalidArgument,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := mute(t, &stubUseCase{err: err}, &chatv1.MuteRequest{AccountId: "not-an-account"})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
