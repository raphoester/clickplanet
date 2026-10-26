package listen_for_events_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// Must stay well under Cloudflare's ~125s idle cut, which kills a silent stream.
const DefaultHeartbeat = 30 * time.Second

type UpdatesSubscriber interface {
	Subscribe(ctx context.Context) (<-chan clicks.Change, clicks.MapVersion, error)
}

type Event struct {
	// The change's, or for a heartbeat the last one the stream sent.
	Version       clicks.MapVersion
	Update        *clicks.TileUpdate
	Blast         *clicks.Blast
	Fortification *clicks.Fortification
	Tally         *clicks.Tally
	Heartbeat     bool

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

type Facts interface {
	Tell(fact facts.Fact)
}

func New(subscriber UpdatesSubscriber, heartbeat time.Duration, bonuses BonusFeed, facts Facts) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{
		subscriber: subscriber,
		heartbeat:  heartbeat,
		bonuses:    bonuses,
		facts:      facts,
	}
}

type UseCase struct {
	subscriber UpdatesSubscriber
	heartbeat  time.Duration
	bonuses    BonusFeed
	facts      Facts
}

// A stream lasts hours: it is told when it opens, and an open that fails opened all the same.
func (u *UseCase) Execute(ctx context.Context, sink Sink) error {
	scope := cpctx.RateLimitKey(ctx)
	u.facts.Tell(func(at time.Time) proto.Message {
		return &planetv1.StreamOpened{Scope: scope, OpenedAt: timestamppb.New(at)}
	})

	updates, version, err := u.subscriber.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to tile updates: %w", err)
	}

	boxes, leave := u.bonuses.Attend(bonuses.EntrantOf(clicks.PayerOf(ctx)))
	defer leave()

	// Connect writes nothing until the first Send, and the client resyncs on a stream's first event.
	if err := sink.Send(Event{Heartbeat: true, Version: version}); err != nil {
		return err
	}

	heartbeat := time.NewTicker(u.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-heartbeat.C:
			if err := sink.Send(Event{Heartbeat: true, Version: version}); err != nil {
				return err
			}

		case change, open := <-updates:
			if !open {
				return nil
			}

			version = change.Version
			if err := sink.Send(Event{
				Version: change.Version, Update: change.Update, Blast: change.Blast,
				Fortification: change.Fortification, Tally: change.Tally,
			}); err != nil {
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
