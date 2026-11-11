package listen_for_events_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
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
	Enclosure     *clicks.Enclosure
	Yours         bool
	Fortification *clicks.Fortification
	KnockOut      *clicks.KnockOut
	Tally         *clicks.Tally
	Heartbeat     bool
	RulesChanged  bool

	Spread  *notices.Spread
	Granted *charges.Held
}

// Notices is planet's own news for the caller: every spread, and what its account holds after a grant.
type Notices interface {
	Attend(caller clicks.Caller) (<-chan notices.Event, func())
}

type RulesFeed interface {
	Watch() (<-chan cprules.Set[*planetv1.Rules], func())
}

type Sink interface {
	Send(event Event) error
}

type Facts interface {
	Tell(fact facts.Fact)
}

func New(
	subscriber UpdatesSubscriber, heartbeat time.Duration, notices Notices, rules RulesFeed, facts Facts,
) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{
		subscriber: subscriber,
		heartbeat:  heartbeat,
		notices:    notices,
		rules:      rules,
		facts:      facts,
	}
}

type UseCase struct {
	subscriber UpdatesSubscriber
	heartbeat  time.Duration
	notices    Notices
	rules      RulesFeed
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

	payer := clicks.PayerOf(ctx)
	caller := clicks.CallerOf(payer)

	news, stopHearing := u.notices.Attend(caller)
	defer stopHearing()

	changes, stopWatching := u.rules.Watch()
	defer stopWatching()

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
				Version: change.Version, Update: change.Update, Blast: change.Blast, Enclosure: change.Enclosure,
				Yours:         change.Enclosure != nil && clicks.CallerOf(change.Enclosure.By) == caller,
				Fortification: change.Fortification, KnockOut: change.KnockOut, Tally: change.Tally,
			}); err != nil {
				return err
			}

		case <-changes:
			if err := sink.Send(Event{RulesChanged: true}); err != nil {
				return err
			}

		case notice := <-news:
			if err := sink.Send(Event{Spread: notice.Spread, Granted: notice.Granted}); err != nil {
				return err
			}
		}
	}
}
