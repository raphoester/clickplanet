package schedule_post_usecase

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// What the service holds, and the posts made here that it may not list yet.
type Feeds interface {
	Posted(ctx context.Context, since time.Time) (calendar.Feed, error)
	Remember(made calendar.Posting)
}

type Service interface {
	Schedule(ctx context.Context, post posts.Post) (posts.Scheduled, error)
}

type Executor interface {
	Execute(ctx context.Context, post posts.Post) Outcome
}

// What became of a post: exactly one of them.
type Outcome struct {
	Scheduled *posts.Scheduled
	// Try again then: plan the slot again first.
	Busy *posts.BusyError
	// Why the slot is no longer open: plan it again.
	Taken string
	// Why the service refused it for good.
	Failed string
}

func New(feeds Feeds, service Service, calendar calendar.Calendar, clock cptime.Clock) *UseCase {
	return &UseCase{feeds: feeds, service: service, calendar: calendar, clock: clock}
}

// One post at a time: the slot is checked against what the service holds, and the post made, under one lock, so two
// jobs never both take the last room of a day. The server is one process, so a mutex is the lock.
type UseCase struct {
	mu       sync.Mutex
	feeds    Feeds
	service  Service
	calendar calendar.Calendar
	clock    cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, post posts.Post) Outcome {
	u.mu.Lock()
	defer u.mu.Unlock()

	now := u.clock.Now()
	feed, err := u.feeds.Posted(ctx, calendar.PostedSince(now))
	if err != nil {
		return outcomeOf(fmt.Errorf("failed to read what each platform holds: %w", err))
	}
	if !u.calendar.Open(post.Slot(), feed, now) {
		return Outcome{Taken: "the slot is no longer open: a post took its room or its spacing, or its time is too near"}
	}

	scheduled, err := u.service.Schedule(ctx, post)
	if err != nil {
		return outcomeOf(err)
	}
	u.feeds.Remember(calendar.Posting{ID: string(scheduled.ID), At: post.Slot().At, Platforms: tookOf(post.Slot(), scheduled)})
	return Outcome{Scheduled: &scheduled}
}

func tookOf(slot calendar.Slot, scheduled posts.Scheduled) []calendar.Platform {
	var took []calendar.Platform
	for _, platform := range slot.Platforms {
		if scheduled.Refused[platform] == nil {
			took = append(took, platform)
		}
	}
	return took
}

func outcomeOf(err error) Outcome {
	var busy *posts.BusyError
	if errors.As(err, &busy) {
		return Outcome{Busy: busy}
	}
	return Outcome{Failed: err.Error()}
}
