package send_message_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
)

type stubUseCase struct {
	in  *send_message_usecase.In
	err error
}

func (s stubUseCase) Execute(_ context.Context, in send_message_usecase.In) (messages.Message, error) {
	*s.in = in
	if s.err != nil {
		return messages.Message{}, s.err
	}

	return messages.Message{
		ID:         "message-1",
		SentAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AuthorName: in.AuthorName,
		AuthorTag:  "a1b2c3",
		CountryID:  in.CountryID,
		Text:       in.Text,
	}, nil
}

func send(useCase stubUseCase) (*connect.Response[chatv1.SendMessageResponse], error) {
	req := connect.NewRequest(&chatv1.SendMessageRequest{
		AuthorName: "Bob",
		AuthorId:   "some-uuid",
		CountryId:  "fr",
		Text:       "hello planet",
	})
	req.Header().Set("User-Agent", "curl/8")

	return send_message_handler.New(useCase).SendMessage(context.Background(), req)
}

func TestSendMessageMapsTheRequest(t *testing.T) {
	var in send_message_usecase.In
	_, err := send(stubUseCase{in: &in})
	require.NoError(t, err)

	assert.Equal(t, send_message_usecase.In{
		AuthorName: "Bob",
		AuthorID:   "some-uuid",
		CountryID:  "fr",
		Text:       "hello planet",
		UserAgent:  "curl/8",
	}, in)
}

func TestSendMessageReturnsTheStoredMessage(t *testing.T) {
	res, err := send(stubUseCase{in: &send_message_usecase.In{}})
	require.NoError(t, err)

	message := res.Msg.GetMessage()
	assert.Equal(t, "message-1", message.GetId())
	assert.Equal(t, "Bob", message.GetAuthorName())
	assert.Equal(t, "a1b2c3", message.GetAuthorTag())
	assert.Equal(t, "fr", message.GetCountryId())
	assert.Equal(t, "hello planet", message.GetText())
	assert.Equal(t, int64(1704067200000), message.GetSentAtUnixMs())
}

func TestARefusedMessageDoesNotLeakWhyItWasRefused(t *testing.T) {
	_, err := send(stubUseCase{
		in:  &send_message_usecase.In{},
		err: fmt.Errorf("%w: text: longer than 280 characters", messages.ErrInvalidMessage),
	})

	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), "280")
	assert.NotContains(t, err.Error(), "text:")
}

func TestAnUnexpectedFailureIsLeftForTheErrorNet(t *testing.T) {
	_, err := send(stubUseCase{in: &send_message_usecase.In{}, err: assert.AnError})

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
