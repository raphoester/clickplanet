package unmute_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/unmute_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/unmute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	in    unmute_usecase.In
	until time.Time
	err   error
}

func (s *stubUseCase) Execute(_ context.Context, in unmute_usecase.In) (time.Time, error) {
	s.in = in
	return s.until, s.err
}

const bullyID = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

func unmute(t *testing.T, useCase *stubUseCase, accountID string) (*chatv1.UnmuteResponse, error) {
	t.Helper()

	return unmuteOf(t, useCase, &chatv1.UnmuteRequest{AccountId: accountID})
}

func unmuteOf(t *testing.T, useCase *stubUseCase, req *chatv1.UnmuteRequest) (*chatv1.UnmuteResponse, error) {
	t.Helper()

	return unmuteAs(t, t.Context(), useCase, req)
}

func unmuteAs(t *testing.T, ctx context.Context, useCase *stubUseCase, req *chatv1.UnmuteRequest) (*chatv1.UnmuteResponse, error) {
	t.Helper()

	res, err := unmute_handler.New(useCase).Unmute(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("unmute refused: %w", err)
	}
	return res.Msg, nil
}

func TestTheAccountReachesTheUseCaseAndWhenItsMuteWouldHaveEndedComesBack(t *testing.T) {
	until := time.Date(2026, 10, 30, 21, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{until: until}

	res, err := unmute(t, useCase, bullyID)
	require.NoError(t, err)

	assert.Equal(t, messages.AccountIDOf(bullyID), useCase.in.Account)
	assert.Equal(t, mutes.Chat, useCase.in.Kind, "no kind is the whole chat")
	assert.Equal(t, until, res.GetMutedUntil().AsTime())
}

func TestTheKindAskedReachesTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := unmuteOf(t, useCase, &chatv1.UnmuteRequest{AccountId: bullyID, Kind: chatv1.MuteKind_MUTE_KIND_GIFS})
	require.NoError(t, err)

	assert.Equal(t, mutes.Gifs, useCase.in.Kind)
}

func TestTheReasonAndWhoAsksReachTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}
	operator := messages.AccountIDOf("0190f5c4-1c2b-7a00-8000-000000000009")
	ctx := cpctx.AddOperatorToContext(t.Context(), cpctx.OperatorOf(operator, ""))

	_, err := unmuteAs(t, ctx, useCase, &chatv1.UnmuteRequest{AccountId: bullyID, Reason: "a mistake"})
	require.NoError(t, err)

	assert.Equal(t, "a mistake", useCase.in.Reason)
	assert.Equal(t, moderation.OperatorOf(operator, ""), useCase.in.By)
}

func TestAKindTheProtoDoesNotNameIsInvalid(t *testing.T) {
	_, err := unmuteOf(t, &stubUseCase{}, &chatv1.UnmuteRequest{AccountId: bullyID, Kind: chatv1.MuteKind(7)})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		mutes.ErrNoAccount: connect.CodeInvalidArgument,
		fmt.Errorf("cannot unmute: %w", moderation.ErrInvalidReason): connect.CodeInvalidArgument,
		fmt.Errorf("%w: %s", mutes.ErrNotMuted, bullyID):             connect.CodeNotFound,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := unmute(t, &stubUseCase{err: err}, bullyID)
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
