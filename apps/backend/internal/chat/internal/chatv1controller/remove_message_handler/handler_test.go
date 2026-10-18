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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
)

type stubUseCase struct {
	asked     messages.MessageID
	removedAt time.Time
	err       error
}

func (s *stubUseCase) Execute(_ context.Context, id messages.MessageID) (time.Time, error) {
	s.asked = id
	return s.removedAt, s.err
}

func remove(t *testing.T, useCase *stubUseCase, id string) (*chatv1.RemoveMessageResponse, error) {
	t.Helper()

	res, err := remove_message_handler.New(useCase).
		RemoveMessage(t.Context(), connect.NewRequest(&chatv1.RemoveMessageRequest{MessageId: id}))
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

	assert.Equal(t, messages.MessageID("insult"), useCase.asked)
	assert.Equal(t, at, res.GetRemovedAt().AsTime())
}

func TestNoIDIsRefusedBeforeTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := remove(t, useCase, "")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("%w: %q", removals.ErrUnknownMessage, "pruned"): connect.CodeNotFound,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := remove(t, &stubUseCase{err: err}, "pruned")
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
