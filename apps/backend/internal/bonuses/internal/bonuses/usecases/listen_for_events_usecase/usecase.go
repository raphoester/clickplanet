package listen_for_events_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
)

// Must stay well under Cloudflare's ~125s idle cut, which kills a silent stream.
const DefaultHeartbeat = 30 * time.Second

// Event is one frame of a caller's stream: a box or a quiz offered to it, a catch by anyone, a set taken, a heartbeat.
type Event struct {
	Heartbeat    bool
	RulesChanged bool

	Offer *bonuses.Offer
	Quiz  *bonuses.QuizOffer
	Taken *bonuses.Taken
}

// Offers is the boxes and quizzes for the caller, and every catch. A caller attends while its stream is open: the
// offers go only to a caller who attends.
type Offers interface {
	Attend(entrant bonuses.Entrant) (<-chan bonuses.Event, func())
}

type RulesFeed interface {
	Watch() (<-chan rules.Set, func())
}

type Sink interface {
	Send(event Event) error
}

func New(offers Offers, rules RulesFeed, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{offers: offers, rules: rules, heartbeat: heartbeat}
}

type UseCase struct {
	offers    Offers
	rules     RulesFeed
	heartbeat time.Duration
}

// Execute attends the offers as the caller until the stream ends: a signed-in account as itself, a guest as its
// address, which every guest behind it shares.
func (u *UseCase) Execute(ctx context.Context, caller bonuses.Caller, sink Sink) error {
	offers, leave := u.offers.Attend(bonuses.EntrantOf(caller))
	defer leave()

	changes, stopWatching := u.rules.Watch()
	defer stopWatching()

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

		case <-changes:
			if err := sink.Send(Event{RulesChanged: true}); err != nil {
				return err
			}

		case event, open := <-offers:
			if !open {
				return nil
			}
			if err := sink.Send(Event{Offer: event.Offer, Quiz: event.Quiz, Taken: event.Taken}); err != nil {
				return err
			}
		}
	}
}
