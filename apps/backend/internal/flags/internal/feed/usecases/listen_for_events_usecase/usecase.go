package listen_for_events_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
)

// Must stay well under Cloudflare's ~125s idle cut, which kills a silent stream.
const DefaultHeartbeat = 30 * time.Second

// Event is one frame of a stream: a flag as it is now, a flag gone, or a heartbeat.
type Event struct {
	Heartbeat bool
	Flag      *catalog.Snapshot
	Gone      *catalog.FlagID
}

type Flags interface {
	Attend() (<-chan feed.Told, func())
}

type Sink interface {
	Send(event Event) error
}

func New(flags Flags, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}
	return &UseCase{flags: flags, heartbeat: heartbeat}
}

type UseCase struct {
	flags     Flags
	heartbeat time.Duration
}

func (u *UseCase) Execute(ctx context.Context, sink Sink) error {
	flags, leave := u.flags.Attend()
	defer leave()

	// Connect writes nothing until the first Send, and the client takes a stream's first event as the sign it is live.
	if err := sink.Send(Event{Heartbeat: true}); err != nil {
		return err
	}

	heartbeat := time.NewTicker(u.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			if err := sink.Send(Event{Heartbeat: true}); err != nil {
				return err
			}
		case flag, open := <-flags:
			if !open {
				return nil
			}
			if err := sink.Send(Event{Flag: flag.Flag, Gone: flag.Gone}); err != nil {
				return err
			}
		}
	}
}
