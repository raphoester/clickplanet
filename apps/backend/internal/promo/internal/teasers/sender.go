package teasers

import (
	"context"
	"errors"
	"time"
)

// How many slots in a row may be taken before the post fails: another job posting at the same time, not a loop.
const maxTaken = 3

type Pause interface {
	Wait(ctx context.Context, d time.Duration) error
}

type Backoff []time.Duration

// failed: the tries made so far, from 1. A wait asked for longer than the last of them is not waited.
func (b Backoff) Wait(failed int, asked time.Duration) (time.Duration, bool) {
	if failed < 1 || failed > len(b) || asked > b[len(b)-1] {
		return 0, false
	}
	return max(b[failed-1], asked), true
}

// The server's publishing module.
type Publishing interface {
	// The first slot from notBefore to by at which every one of the platforms takes the post: false when none does.
	Slot(ctx context.Context, platforms []Platform, notBefore, by time.Time) (Slot, bool, error)
	// The post at its slot. Busy (*BusyError): try later. ErrTaken: plan again.
	Schedule(ctx context.Context, post Post) (Receipt, error)
}

func NewSender(publishing Publishing, backoff Backoff, pause Pause) *Sender {
	return &Sender{publishing: publishing, backoff: backoff, pause: pause}
}

type Sender struct {
	publishing Publishing
	backoff    Backoff
	pause      Pause
}

// The post at the first slot of its window that the calendar gives it, planned again after each wait and each slot
// taken, so a wait never sends it at a time already past or crowded. Held, and no error, when no platform takes it
// before its battle. It answers the post as it was sent, at its slot.
func (s *Sender) Send(ctx context.Context, post Post) (sent Post, receipt Receipt, held bool, err error) {
	notBefore, by := post.Teaser.Window()
	busy, taken := 0, 0
	for {
		slot, found, err := s.slot(ctx, notBefore, by)
		if err == nil && !found {
			return post, Receipt{}, true, nil
		}
		if err == nil {
			post.Slot = slot
			var receipt Receipt
			if receipt, err = s.publishing.Schedule(ctx, post); err == nil {
				return post, receipt, false, nil
			}
			if errors.Is(err, ErrTaken) && taken < maxTaken {
				taken++
				continue
			}
		}

		var refusal *BusyError
		if !errors.As(err, &refusal) {
			return post, Receipt{}, false, err //nolint:wrapcheck // a pure delegation: the calendar or the service already named what failed.
		}
		busy++
		wait, again := s.backoff.Wait(busy, refusal.RetryAfter)
		if !again {
			return post, Receipt{}, false, err //nolint:wrapcheck // a pure delegation: the service already named what failed.
		}
		if err := s.pause.Wait(ctx, wait); err != nil {
			return post, Receipt{}, false, err //nolint:wrapcheck // a pure delegation: the pause only ends early with the context.
		}
	}
}

// The slot of the first of the choices that takes the post by its battle.
func (s *Sender) slot(ctx context.Context, notBefore, by time.Time) (Slot, bool, error) {
	for _, platforms := range choices {
		slot, found, err := s.publishing.Slot(ctx, platforms, notBefore, by)
		if err != nil || found {
			return slot, found, err //nolint:wrapcheck // a pure delegation: the calendar already named what failed.
		}
	}
	return Slot{}, false, nil
}

// The pause of a real run.
type SystemPause struct{}

func (SystemPause) Wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // the context's own error says it ended.
	case <-timer.C:
		return nil
	}
}
