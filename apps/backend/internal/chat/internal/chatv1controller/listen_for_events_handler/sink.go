package listen_for_events_handler

import (
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
)

// EventStream is the one method of connect's server stream a sink uses.
type EventStream interface {
	Send(event *chatv1.ChatEvent) error
}

func NewSink(stream EventStream) Sink {
	return Sink{stream: stream}
}

// Sink writes the use case's frames as the proto envelope.
type Sink struct {
	stream EventStream
}

var _ listen_for_events_usecase.Sink = Sink{}

func (s Sink) Send(event listen_for_events_usecase.Event) error {
	if event.Heartbeat {
		return s.stream.Send(&chatv1.ChatEvent{
			Event: &chatv1.ChatEvent_Heartbeat{Heartbeat: &chatv1.Heartbeat{}},
		})
	}

	if tally := event.Update.Reactions; tally != nil {
		return s.stream.Send(&chatv1.ChatEvent{
			Event: &chatv1.ChatEvent_Reactions{Reactions: &chatv1.ReactionsChanged{
				MessageId: string(tally.MessageID),
				Reactions: chatmessage.EncodeCounts(tally.Counts),
			}},
		})
	}

	return s.stream.Send(&chatv1.ChatEvent{
		Event: &chatv1.ChatEvent_Message{Message: chatmessage.Encode(*event.Update.Message, nil)},
	})
}
