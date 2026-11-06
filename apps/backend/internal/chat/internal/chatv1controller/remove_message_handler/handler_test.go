package remove_message_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/remove_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	asked     remove_message_usecase.In
	removedAt time.Time
	err       error
}

func (s *stubUseCase) Execute(_ context.Context, in remove_message_usecase.In) (time.Time, error) {
	s.asked = in
	return s.removedAt, s.err
}

func remove(t *testing.T, useCase *stubUseCase, id string) (*chatv1.RemoveMessageResponse, error) {
	t.Helper()

	return removeAs(t, t.Context(), useCase, &chatv1.RemoveMessageRequest{MessageId: id})
}

func removeAs(
	t *testing.T, ctx context.Context, useCase *stubUseCase, req *chatv1.RemoveMessageRequest,
) (*chatv1.RemoveMessageResponse, error) {
	t.Helper()

	res, err := remove_message_handler.New(useCase).RemoveMessage(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("removal refused: %w", err)
	}
	return res.Msg, nil
}

func TestTheMessageIsRemovedAndWhenComesBack(t *testing.T) {
	at := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{removedAt: at}

	res, err := remove(t, useCase, "insult")
	require.NoError(t, err)

	assert.Equal(t, messages.MessageID("insult"), useCase.asked.Message)
	assert.Equal(t, at, res.GetRemovedAt().AsTime())
}

func TestTheReasonAndWhoAsksReachTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}
	operator := messages.AccountIDOf("0190f5c4-1c2b-7a00-8000-000000000009")
	ctx := cpctx.AddOperatorToContext(t.Context(), cpctx.OperatorOf(operator, "Claude"))

	_, err := removeAs(t, ctx, useCase, &chatv1.RemoveMessageRequest{MessageId: "insult", Reason: "spam"})
	require.NoError(t, err)

	assert.Equal(t, "spam", useCase.asked.Reason)
	assert.Equal(t, moderation.OperatorOf(operator, "Claude"), useCase.asked.By)
}

func TestNoIDIsRefusedBeforeTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := remove(t, useCase, "")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, useCase.asked.Message)
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("%w: %q", removals.ErrUnknownMessage, "pruned"):               connect.CodeNotFound,
		fmt.Errorf("cannot remove the message: %w", moderation.ErrInvalidReason): connect.CodeInvalidArgument,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := remove(t, &stubUseCase{err: err}, "pruned")
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
