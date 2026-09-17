package listen_for_events_handler

import (
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
)

// EventStream is the one method of connect's server stream a sink uses.
type EventStream interface {
	Send(event *playerv1.PlayerEvent) error
}

func NewSink(stream EventStream) Sink {
	return Sink{stream: stream}
}

// Sink writes the use case's frames as the proto envelope.
type Sink struct {
	stream EventStream
}

var _ listen_for_events_usecase.Sink = Sink{}

func (s Sink) SendRoster(roster []presence.Entry) error {
	return s.stream.Send(&playerv1.PlayerEvent{
		Event: &playerv1.PlayerEvent_Roster{Roster: &playerv1.Roster{Entries: playermessage.RosterEntries(roster)}},
	})
}

func (s Sink) SendChange(change presence.Change) error {
	if change.Left {
		return s.stream.Send(&playerv1.PlayerEvent{
			Event: &playerv1.PlayerEvent_Left{Left: &playerv1.PlayerLeft{Key: string(change.Entry.Key)}},
		})
	}

	return s.stream.Send(&playerv1.PlayerEvent{
		Event: &playerv1.PlayerEvent_Entry{Entry: playermessage.RosterEntry(change.Entry)},
	})
}

func (s Sink) SendHeartbeat() error {
	return s.stream.Send(&playerv1.PlayerEvent{
		Event: &playerv1.PlayerEvent_Heartbeat{Heartbeat: &playerv1.Heartbeat{}},
	})
}
