package listen_for_events_handler

import (
	"errors"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatannouncement"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
)

type EventStream interface {
	Send(event *chatv1.ChatEvent) error
}

var errEmptyUpdate = errors.New("a chat update with nothing in it")

func NewSink(stream EventStream) Sink {
	return Sink{stream: stream}
}

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

	if tally, changed := event.Update.Reactions(); changed {
		return s.stream.Send(&chatv1.ChatEvent{
			Event: &chatv1.ChatEvent_Reactions{Reactions: &chatv1.ReactionsChanged{
				MessageId: string(tally.MessageID()),
				Reactions: chatmessage.EncodeCounts(tally.Counts()),
				Version:   tally.Version(),
			}},
		})
	}

	if announcement, announced := event.Update.Announcement(); announced {
		return s.stream.Send(&chatv1.ChatEvent{
			Event: &chatv1.ChatEvent_Announcement{Announcement: chatannouncement.Encode(announcement)},
		})
	}

	message, sent := event.Update.Message()
	if !sent {
		return errEmptyUpdate
	}
	return s.stream.Send(&chatv1.ChatEvent{
		Event: &chatv1.ChatEvent_Message{Message: chatmessage.Encode(message, nil, 0)},
	})
}
