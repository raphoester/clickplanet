// Package listen_for_events_usecase runs one client's live roster: the whole roster, then each change.
package listen_for_events_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
)

// DefaultHeartbeat is well under Cloudflare's ~125s idle cut.
const DefaultHeartbeat = 30 * time.Second

type VisitsSubscriber interface {
	Subscribe(ctx context.Context) ([]presence.Entry, <-chan presence.Change)
}

// Sink carries each frame to the caller.
type Sink interface {
	SendRoster(roster []presence.Entry) error
	SendChange(change presence.Change) error
	SendHeartbeat() error
}

func New(subscriber VisitsSubscriber, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{subscriber: subscriber, heartbeat: heartbeat}
}

type UseCase struct {
	subscriber VisitsSubscriber
	heartbeat  time.Duration
}

// Execute returns when the context ends, or when the storage closed the feed of a caller that fell behind.
func (u *UseCase) Execute(ctx context.Context, sink Sink) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	roster, changes := u.subscriber.Subscribe(ctx)
	if err := sink.SendRoster(roster); err != nil {
		return err //nolint:wrapcheck // the stream's own error.
	}

	heartbeat := time.NewTicker(u.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-heartbeat.C:
			if err := sink.SendHeartbeat(); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}

		case change, open := <-changes:
			if !open {
				return nil
			}

			if err := sink.SendChange(change); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}
		}
	}
}
