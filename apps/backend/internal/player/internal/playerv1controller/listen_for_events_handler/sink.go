package listen_for_events_handler

import (
	"sync"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/listen_for_titles_usecase"
)

type EventStream interface {
	Send(event *playerv1.PlayerEvent) error
}

func NewSink(stream EventStream) *Sink {
	return &Sink{stream: stream}
}

type Sink struct {
	mu     sync.Mutex
	stream EventStream
}

var (
	_ listen_for_events_usecase.Sink = (*Sink)(nil)
	_ listen_for_titles_usecase.Sink = (*Sink)(nil)
)

func (s *Sink) send(event *playerv1.PlayerEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.stream.Send(event)
}

func (s *Sink) SendRoster(roster []presence.Entry) error {
	return s.send(&playerv1.PlayerEvent{
		Event: &playerv1.PlayerEvent_Roster{Roster: &playerv1.Roster{Entries: playermessage.RosterEntries(roster)}},
	})
}

func (s *Sink) SendChange(change presence.Change) error {
	if change.Left() {
		return s.send(&playerv1.PlayerEvent{
			Event: &playerv1.PlayerEvent_Left{Left: &playerv1.PlayerLeft{Key: string(change.Entry().Key())}},
		})
	}

	return s.send(&playerv1.PlayerEvent{
		Event: &playerv1.PlayerEvent_Entry{Entry: playermessage.RosterEntry(change.Entry())},
	})
}

func (s *Sink) SendHeartbeat() error {
	return s.send(&playerv1.PlayerEvent{
		Event: &playerv1.PlayerEvent_Heartbeat{Heartbeat: &playerv1.Heartbeat{}},
	})
}

func (s *Sink) SendTitleEarned(standing titles.Standing) error {
	return s.send(&playerv1.PlayerEvent{
		Event: &playerv1.PlayerEvent_TitleEarned{TitleEarned: &playerv1.TitleEarned{Title: playermessage.Title(standing)}},
	})
}
