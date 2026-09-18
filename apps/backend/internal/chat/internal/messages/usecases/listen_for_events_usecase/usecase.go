// Package listen_for_events_usecase runs one client's live chat feed, heartbeat included.
package listen_for_events_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// DefaultHeartbeat is well under Cloudflare's ~125s idle cut.
const DefaultHeartbeat = 30 * time.Second

type MessagesSubscriber interface {
	Subscribe(ctx context.Context) (<-chan messages.Update, error)
}

// Event is one frame of the feed: an update, or a heartbeat.
type Event struct {
	Update    messages.Update
	Heartbeat bool
}

// Sink carries a frame to the caller.
type Sink interface {
	Send(event Event) error
}

func New(subscriber MessagesSubscriber, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{subscriber: subscriber, heartbeat: heartbeat}
}

type UseCase struct {
	subscriber MessagesSubscriber
	heartbeat  time.Duration
}

// Execute returns when the context ends; cancelling it is what unsubscribes.
func (u *UseCase) Execute(ctx context.Context, sink Sink) error {
	feed, err := u.subscriber.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to chat messages: %w", err)
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
