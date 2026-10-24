package posts

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

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

func NewSender(service Service, backoff Backoff, pause Pause, clock cptime.Clock) *Sender {
	return &Sender{service: service, backoff: backoff, pause: pause, clock: clock}
}

type Sender struct {
	service Service
	backoff Backoff
	pause   Pause
	clock   cptime.Clock
}

// It answers the post as it was sent: a wait that ate its lead moves it later, never sooner.
func (s *Sender) Send(ctx context.Context, post Post) (Post, Scheduled, error) {
	for failed := 1; ; failed++ {
		post = post.Rescheduled(s.clock.Now())

		scheduled, err := s.service.Schedule(ctx, post)
		var busy *BusyError
		if !errors.As(err, &busy) {
			return post, scheduled, err //nolint:wrapcheck // a pure delegation: the service already named what failed.
		}

		wait, again := s.backoff.Wait(failed, busy.RetryAfter)
		if !again {
			return post, Scheduled{}, err //nolint:wrapcheck // a pure delegation: the service already named what failed.
		}
		if err := s.pause.Wait(ctx, wait); err != nil {
			return post, Scheduled{}, err //nolint:wrapcheck // a pure delegation: the pause only ends early with the context.
		}
	}
}
