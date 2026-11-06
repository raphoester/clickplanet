package remove_messages_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/remove_messages_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_messages_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	asked     *remove_messages_usecase.In
	removedAt time.Time
	err       error
}

func (s *stubUseCase) Execute(_ context.Context, in remove_messages_usecase.In) (time.Time, error) {
	s.asked = &in
	return s.removedAt, s.err
}

func removeAs(
	t *testing.T, ctx context.Context, useCase *stubUseCase, removals ...*chatv1.MessageRemoval,
) (*chatv1.RemoveMessagesResponse, error) {
	t.Helper()

	res, err := remove_messages_handler.New(useCase).RemoveMessages(ctx,
		connect.NewRequest(&chatv1.RemoveMessagesRequest{Removals: removals}))
	if err != nil {
		return nil, fmt.Errorf("removals refused: %w", err)
	}
	return res.Msg, nil
}

func TestEveryMessageAndItsReasonReachTheUseCaseWithWhoAsks(t *testing.T) {
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{removedAt: at}
	operator := messages.AccountIDOf("0190f5c4-1c2b-7a00-8000-000000000009")
	ctx := cpctx.AddOperatorToContext(t.Context(), cpctx.OperatorOf(operator, "Claude"))

	res, err := removeAs(t, ctx, useCase,
		&chatv1.MessageRemoval{MessageId: "insult", Reason: "offence 1 (chat): an insult"},
		&chatv1.MessageRemoval{MessageId: "spam", Reason: "spam, no offence"},
	)
	require.NoError(t, err)

	require.NotNil(t, useCase.asked)
	assert.Equal(t, []remove_messages_usecase.Removal{
		{Message: "insult", Reason: "offence 1 (chat): an insult"},
		{Message: "spam", Reason: "spam, no offence"},
	}, useCase.asked.Removals)
	assert.Equal(t, moderation.OperatorOf(operator, "Claude"), useCase.asked.By)
	assert.Equal(t, at, res.GetRemovedAt().AsTime())
}

func TestAnEmptyIDIsRefusedBeforeTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := removeAs(t, t.Context(), useCase,
		&chatv1.MessageRemoval{MessageId: "insult"}, &chatv1.MessageRemoval{Reason: "spam"})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Contains(t, err.Error(), "removal 1")
	assert.Nil(t, useCase.asked)
}

func TestErrorsMapToTheirCodesAndNameTheMessage(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("%w: %q", removals.ErrUnknownMessage, "pruned"):                         connect.CodeNotFound,
		fmt.Errorf("message %q: %w", "pruned", moderation.ErrInvalidReason):                connect.CodeInvalidArgument,
		fmt.Errorf("%w: %q", remove_messages_usecase.ErrRemovedTwice, "pruned"):            connect.CodeInvalidArgument,
		fmt.Errorf("cannot remove the messages: %w", remove_messages_usecase.ErrNoRemoval): connect.CodeInvalidArgument,
		remove_messages_usecase.ErrTooManyRemovals:                                         connect.CodeInvalidArgument,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := removeAs(t, t.Context(), &stubUseCase{err: err}, &chatv1.MessageRemoval{MessageId: "pruned"})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
		if code != connect.CodeUnknown {
			assert.Contains(t, got.Error(), err.Error(), "the refusal says what the use case said")
		}
	}
}
