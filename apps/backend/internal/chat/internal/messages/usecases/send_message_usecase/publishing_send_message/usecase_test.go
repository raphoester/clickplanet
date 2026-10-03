package publishing_send_message_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/publishing_send_message"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type stubUseCase struct {
	message messages.Message
	err     error
}

func (s stubUseCase) Execute(context.Context, send_message_usecase.In) (messages.Message, error) {
	return s.message, s.err
}

var sent = messages.Message{
	ID:      "6b1f0c4e-6d0e-4b8f-9d55-2d3c4a1e0f77",
	SentAt:  time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC),
	Account: messages.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"),
	Text:    "hello",
}

func TestASentMessageIsPublishedWithItsSenderAndTime(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()

	message, err := publishing_send_message.New(stubUseCase{message: sent}, events).
		Execute(t.Context(), send_message_usecase.In{Account: sent.Account, Text: sent.Text})

	require.NoError(t, err)
	assert.Equal(t, sent, message)
	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&chatv1.MessageSent{
		MessageId: string(sent.ID), AccountId: sent.Account.String(), SentAt: timestamppb.New(sent.SentAt),
	}, events.Published()[0]))
}

func TestARefusedMessagePublishesNothing(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()
	refused := errors.New("postgres is down")

	_, err := publishing_send_message.New(stubUseCase{err: refused}, events).
		Execute(t.Context(), send_message_usecase.In{Account: sent.Account, Text: sent.Text})

	require.ErrorIs(t, err, refused)
	assert.Empty(t, events.Published())
}
