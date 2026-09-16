package listen_for_events_handler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/listen_for_events_usecase"
)

type recorder struct {
	sent []*chatv1.ChatEvent
	err  error
}

func (r *recorder) Send(event *chatv1.ChatEvent) error {
	r.sent = append(r.sent, event)
	return r.err
}

func TestSinkFramesAMessage(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Message: &messages.Message{ID: "message-1", AuthorName: "Bob", Text: "hello"},
	})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)

	message := stream.sent[0].GetMessage()
	require.NotNil(t, message, "a message travels as the message case")
	assert.Nil(t, stream.sent[0].GetHeartbeat())
	assert.Equal(t, "message-1", message.GetId())
	assert.Equal(t, "Bob", message.GetAuthorName())
	assert.Equal(t, "hello", message.GetText())
}

func TestSinkFramesAHeartbeat(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{Heartbeat: true})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)
	require.NotNil(t, stream.sent[0].GetHeartbeat(), "a heartbeat travels as the heartbeat case")
	assert.Nil(t, stream.sent[0].GetMessage())
}

func TestSinkReportsAFailedSend(t *testing.T) {
	err := listen_for_events_handler.NewSink(&recorder{err: assert.AnError}).
		Send(listen_for_events_usecase.Event{Heartbeat: true})

	require.ErrorIs(t, err, assert.AnError)
}

func TestSinkFramesARedaction(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Redaction: &messages.Redaction{AuthorTag: "a1b2c3"},
	})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)

	redacted := stream.sent[0].GetMemberRedacted()
	require.NotNil(t, redacted, "a redaction travels as its own case")
	assert.Nil(t, stream.sent[0].GetMessage())
	assert.Equal(t, "a1b2c3", redacted.GetAuthorTag())
}
