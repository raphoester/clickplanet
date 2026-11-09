package posts

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

// failed: the tries made so far, from 1.
func (b Backoff) Wait(failed int, asked time.Duration) (time.Duration, bool) {
	if failed < 1 || failed > len(b) || asked > b[len(b)-1] {
		return 0, false
	}
	return max(b[failed-1], asked), true
}

// The publishing module's calendar.
type Planner interface {
	// The slot of a post on the platforms from notBefore on, zero for none of the run's own: false when none takes it.
	Slot(ctx context.Context, platforms []Platform, notBefore time.Time) (Slot, bool, error)
}

func NewSender(planner Planner, service Service, backoff Backoff, pause Pause) *Sender {
	return &Sender{planner: planner, service: service, backoff: backoff, pause: pause}
}

type Sender struct {
	planner Planner
	service Service
	backoff Backoff
	pause   Pause
}

// The post at the slot the calendar gives it, planned again after each wait and each slot taken, so a wait never sends
// it at a time already past or crowded. Held, and no error, when no platform takes it. It answers the post as it was
// sent: its slot, or the platforms asked when none was given.
func (s *Sender) Send(ctx context.Context, post Post, notBefore time.Time) (sent Post, scheduled Scheduled, held bool, err error) {
	asked := post.platforms
	busy, taken := 0, 0
	for {
		slot, found, err := s.planner.Slot(ctx, asked, notBefore)
		if err == nil && !found {
			return post, Scheduled{}, true, nil
		}
		if err == nil {
			post = post.At(slot)
			var scheduled Scheduled
			if scheduled, err = s.service.Schedule(ctx, post); err == nil {
				return post, scheduled, false, nil
			}
			if errors.Is(err, ErrTaken) && taken < maxTaken {
				taken++
				continue
			}
		}

		var refusal *BusyError
		if !errors.As(err, &refusal) {
			return post, Scheduled{}, false, err //nolint:wrapcheck // a pure delegation: the calendar or the service already named what failed.
		}
		busy++
		wait, again := s.backoff.Wait(busy, refusal.RetryAfter)
		if !again {
			return post, Scheduled{}, false, err //nolint:wrapcheck // a pure delegation: the service already named what failed.
		}
		if err := s.pause.Wait(ctx, wait); err != nil {
			return post, Scheduled{}, false, err //nolint:wrapcheck // a pure delegation: the pause only ends early with the context.
		}
	}
}
