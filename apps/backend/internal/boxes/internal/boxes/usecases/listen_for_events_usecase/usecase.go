package listen_for_events_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
)

// Must stay well under Cloudflare's ~125s idle cut, which kills a silent stream.
const DefaultHeartbeat = 30 * time.Second

// Event is one frame of a caller's stream: a box offered to it, or a heartbeat.
type Event struct {
	Heartbeat bool

	Offer *boxes.Offer
}

// Boxes is the boxes for the caller. A caller attends while its stream is open: a box goes only to a caller who attends.
type Boxes interface {
	Attend(entrant boxes.Entrant) (<-chan boxes.Offer, func())
}

type Sink interface {
	Send(event Event) error
}

func New(boxes Boxes, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{boxes: boxes, heartbeat: heartbeat}
}

type UseCase struct {
	boxes     Boxes
	heartbeat time.Duration
}

// Execute attends the boxes as the caller until the stream ends: a signed-in account as itself, a guest as its
// address, which every guest behind it shares.
func (u *UseCase) Execute(ctx context.Context, caller boxes.Caller, sink Sink) error {
	offers, leave := u.boxes.Attend(boxes.EntrantOf(caller))
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

		case offer, open := <-offers:
			if !open {
				return nil
			}
			if err := sink.Send(Event{Offer: &offer}); err != nil {
				return err
			}
		}
	}
}
