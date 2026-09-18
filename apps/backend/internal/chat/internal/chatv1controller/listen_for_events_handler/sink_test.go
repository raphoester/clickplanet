package listen_for_events_handler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
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
		Update: feed.Update{Message: &messages.Message{ID: "message-1", AuthorName: "Bob", Text: "hello"}},
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

func TestSinkFramesNewReactions(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Update: feed.Update{Reactions: &reactions.Tally{
			MessageID: "message-1",
			Counts:    []reactions.Count{{Reaction: reactions.Reaction(chatv1.Reaction_REACTION_SKULL), Count: 2}},
		}},
	})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)

	reactions := stream.sent[0].GetReactions()
	require.NotNil(t, reactions, "reactions travel as the reactions case")
	assert.Nil(t, stream.sent[0].GetMessage())
	assert.Equal(t, "message-1", reactions.GetMessageId())
	require.Len(t, reactions.GetReactions(), 1)
	assert.Equal(t, chatv1.Reaction_REACTION_SKULL, reactions.GetReactions()[0].GetReaction())
	assert.Equal(t, uint32(2), reactions.GetReactions()[0].GetCount())
	assert.False(t, reactions.GetReactions()[0].GetMine(), "the stream is nobody's")
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
