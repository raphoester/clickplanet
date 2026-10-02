package listen_for_events_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
)

// Well under Cloudflare's ~125s idle cut on a silent stream.
const DefaultHeartbeat = 30 * time.Second

type UpdatesSubscriber interface {
	Subscribe(ctx context.Context) (<-chan feed.Update, error)
}

type Event struct {
	Update    feed.Update
	Heartbeat bool
}

type Sink interface {
	Send(event Event) error
}

func New(subscriber UpdatesSubscriber, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{subscriber: subscriber, heartbeat: heartbeat}
}

type UseCase struct {
	subscriber UpdatesSubscriber
	heartbeat  time.Duration
}

func (u *UseCase) Execute(ctx context.Context, sink Sink) error {
	feed, err := u.subscriber.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to the chat feed: %w", err)
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

		case update, open := <-feed:
			if !open {
				return nil
			}

			if err := sink.Send(Event{Update: update}); err != nil {
				return err
			}
		}
	}
}
