package listen_for_events_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
)

// Must stay well under Cloudflare's ~125s idle cut, which kills a silent stream.
const DefaultHeartbeat = 30 * time.Second

// Event is one frame of a caller's stream: a quiz offered to it, a quiz won by anyone, a heartbeat.
type Event struct {
	Heartbeat bool

	Offer *quizzes.Offer
	Won   *quizzes.Won
}

// Quizzes is the quizzes for the caller, and every quiz won. A caller attends while its stream is open: a quiz goes only
// to a caller who attends.
type Quizzes interface {
	Attend(entrant quizzes.Entrant) (<-chan quizzes.Event, func())
}

type Sink interface {
	Send(event Event) error
}

func New(quizzes Quizzes, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{quizzes: quizzes, heartbeat: heartbeat}
}

type UseCase struct {
	quizzes   Quizzes
	heartbeat time.Duration
}

// Execute attends the quizzes as the caller until the stream ends: a signed-in account as itself, a guest as its
// address, which every guest behind it shares.
func (u *UseCase) Execute(ctx context.Context, caller quizzes.Caller, sink Sink) error {
	events, leave := u.quizzes.Attend(quizzes.EntrantOf(caller))
	defer leave()

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

		case event, open := <-events:
			if !open {
				return nil
			}
			if err := sink.Send(Event{Offer: event.Offer, Won: event.Won}); err != nil {
				return err
			}
		}
	}
}
