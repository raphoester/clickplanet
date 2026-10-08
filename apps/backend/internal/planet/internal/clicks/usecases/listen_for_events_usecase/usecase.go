package listen_for_events_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Must stay well under Cloudflare's ~125s idle cut, which kills a silent stream.
const DefaultHeartbeat = 30 * time.Second

type UpdatesSubscriber interface {
	Subscribe(ctx context.Context) (<-chan clicks.Change, error)
}

type Event struct {
	Update    clicks.TileUpdate
	Blast     *clicks.Blast
	Heartbeat bool

	Offer    *bonuses.Offer
	Quiz     *bonuses.QuizOffer
	Taken    *bonuses.Taken
	Enclosed *bonuses.Enclosed
	Spread   *bonuses.Spread
}

type BonusFeed interface {
	Attend(entrant bonuses.Entrant) (<-chan bonuses.Event, func())
}

type Sink interface {
	Send(event Event) error
}

func New(subscriber UpdatesSubscriber, heartbeat time.Duration, bonuses BonusFeed) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{
		subscriber: subscriber,
		heartbeat:  heartbeat,
		bonuses:    bonuses,
	}
}

type UseCase struct {
	subscriber UpdatesSubscriber
	heartbeat  time.Duration
	bonuses    BonusFeed
}

func (u *UseCase) Execute(ctx context.Context, sink Sink) error {
	updates, err := u.subscriber.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to tile updates: %w", err)
	}

	boxes, leave := u.bonuses.Attend(bonuses.EntrantOf(clicks.PayerOf(ctx)))
	defer leave()

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

		case change, open := <-updates:
			if !open {
				return nil
			}

			if err := sink.Send(eventOf(change)); err != nil {
				return err
			}

		case event, open := <-boxes:
			if !open {
				return nil
			}

			if err := sink.Send(Event{
				Offer: event.Offer, Quiz: event.Quiz, Taken: event.Taken, Enclosed: event.Enclosed,
				Spread: event.Spread,
			}); err != nil {
				return err
			}
		}
	}
}

func eventOf(change clicks.Change) Event {
	if change.Blast != nil {
		return Event{Blast: change.Blast}
	}

	return Event{Update: *change.Update}
}
