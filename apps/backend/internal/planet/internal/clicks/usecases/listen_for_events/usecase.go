// Package listen_for_events runs one client's live feed of the map: every tile
// update it is sent, and the heartbeat that keeps a silent feed from being cut.
package listen_for_events

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// DefaultHeartbeat is well under Cloudflare's ~125s idle cut, and cheap: a
// heartbeat is two bytes.
const DefaultHeartbeat = 30 * time.Second

type UpdatesSubscriber interface {
	Subscribe(ctx context.Context) (<-chan clicks.TileUpdate, error)
}

// Event is one frame of the feed. Exactly one of the two cases is set, which is
// the same shape the wire has — a new kind of frame is a new case here, never a
// second feed.
type Event struct {
	Update    clicks.TileUpdate
	Heartbeat bool
}

// Sink is whatever carries a frame to the caller. The use case decides what to
// send and when; how a frame is written down is the edge's business.
type Sink interface {
	Send(event Event) error
}

func New(subscriber UpdatesSubscriber, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{
		subscriber: subscriber,
		heartbeat:  heartbeat,
	}
}

type UseCase struct {
	subscriber UpdatesSubscriber
	heartbeat  time.Duration
}

// Execute returns when the feed ends, and the context is what ends it: it is
// cancelled however the caller goes away, and cancelling it is what unsubscribes.
func (u *UseCase) Execute(ctx context.Context, sink Sink) error {
	updates, err := u.subscriber.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to tile updates: %w", err)
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

		case update, open := <-updates:
			if !open {
				return nil
			}

			if err := sink.Send(Event{Update: update}); err != nil {
				return err
			}
		}
	}
}
