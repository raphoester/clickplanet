package listen_for_events_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// Well under Cloudflare's ~125s idle cut on a silent stream.
const DefaultHeartbeat = 30 * time.Second

type UpdatesSubscriber interface {
	Subscribe(ctx context.Context) (<-chan feed.Update, error)
}

type Rooms interface {
	EntryError(ctx context.Context, account messages.AccountID, room messages.Room) error
}

type In struct {
	Viewer messages.AccountID
	Room   messages.Room
}

type Event struct {
	Update    feed.Update
	Heartbeat bool
}

type Sink interface {
	Send(event Event) error
}

func New(subscriber UpdatesSubscriber, rooms Rooms, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{subscriber: subscriber, rooms: rooms, heartbeat: heartbeat}
}

type UseCase struct {
	subscriber UpdatesSubscriber
	rooms      Rooms
	heartbeat  time.Duration
}

func (u *UseCase) Execute(ctx context.Context, in In, sink Sink) error {
	feed, err := u.subscriber.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to the chat feed: %w", err)
	}

	// After the subscription: a flag pledged in between is either read here or heard on the feed.
	if err := u.rooms.EntryError(ctx, in.Viewer, in.Room); err != nil {
		return fmt.Errorf("failed to enter the room: %w", err)
	}

	// Connect writes nothing until the first Send, and the client resyncs on a stream's first event.
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

		case update, open := <-feed:
			if !open {
				return nil
			}
			if update.Leaves(in.Viewer, in.Room) {
				return fmt.Errorf("%w: it pledged another flag", messages.ErrNotInRoom)
			}
			if !update.For(in.Room) {
				continue
			}

			if err := sink.Send(Event{Update: update}); err != nil {
				return err
			}
		}
	}
}
