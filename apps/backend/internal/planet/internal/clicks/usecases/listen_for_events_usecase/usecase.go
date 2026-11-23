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

// UpdatesSubscriber is the map that plays: a change of its stage starts a new epoch and cuts every stream, so a stage
// read once subscribed is the stage of every change the stream hears.
type UpdatesSubscriber interface {
	Subscribe(ctx context.Context) (<-chan clicks.Change, clicks.MapVersion, error)
	Stage() clicks.Stage
}

// Boards names a board by its row.
type Boards interface {
	Name(id clicks.BoardID) clicks.BoardName
}

type Event struct {
	// The change's, or for a heartbeat the last one the stream sent.
	Version clicks.MapVersion
	// The board the map plays, on the heartbeat the stream opens with.
	Board         clicks.BoardName
	Update        *clicks.TileUpdate
	Blast         *clicks.Blast
	Enclosure     *clicks.Enclosure
	Yours         bool
	Fortification *clicks.Fortification
	KnockOut      *clicks.KnockOut
	CutOff        *clicks.CutOff
	Infection     *clicks.Infection
	Explosion     *clicks.Explosion
	Tally         *clicks.Tally
	Heartbeat     bool
	RulesChanged  bool
	FlagsChanged  bool

	Spread   *notices.Spread
	Granted  *charges.Gain
	MineLaid *notices.MineLaid
	Rally    *notices.RallyCalled
}

// Notices is planet's own news for the caller: every spread, what a grant added to its account's hand, and the mines its
// flag's players laid.
type Notices interface {
	Attend(holder charges.Holder, flag string) (<-chan notices.Event, func())
}

// Flags is the flag the caller plays for as its stream opens: the mines it hears are that flag's.
type Flags interface {
	FlagOf(ctx context.Context, payer clicks.Payer) (string, error)
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
	subscriber UpdatesSubscriber, boards Boards, heartbeat time.Duration, notices Notices, flags Flags, rules RulesFeed,
	facts Facts,
) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{
		subscriber: subscriber,
		boards:     boards,
		heartbeat:  heartbeat,
		notices:    notices,
		flags:      flags,
		rules:      rules,
		facts:      facts,
	}
}

type UseCase struct {
	subscriber UpdatesSubscriber
	boards     Boards
	heartbeat  time.Duration
	notices    Notices
	flags      Flags
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
	board := u.boards.Name(u.subscriber.Stage().Board)

	payer := clicks.PayerOf(ctx)
	caller := clicks.CallerOf(payer)

	// A flag that could not be read hears no mine until the next pledge: the page reads them with GetMines.
	flag, _ := u.flags.FlagOf(ctx, payer)
	news, stopHearing := u.notices.Attend(charges.HolderOf(payer), flag)
	defer stopHearing()

	changes, stopWatching := u.rules.Watch()
	defer stopWatching()

	// Connect writes nothing until the first Send, and the client resyncs on a stream's first event.
	if err := sink.Send(Event{Heartbeat: true, Version: version, Board: board}); err != nil {
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
				Fortification: change.Fortification, KnockOut: change.KnockOut, CutOff: change.CutOff,
				Infection: change.Infection, Explosion: change.Explosion, Tally: change.Tally,
			}); err != nil {
				return err
			}

		case <-changes:
			if err := sink.Send(Event{RulesChanged: true}); err != nil {
				return err
			}

		case notice := <-news:
			if err := sink.Send(Event{
				Spread: notice.Spread, Granted: notice.Granted, MineLaid: notice.MineLaid, Rally: notice.Rally,
				FlagsChanged: notice.FlagsChanged,
			}); err != nil {
				return err
			}
		}
	}
}
