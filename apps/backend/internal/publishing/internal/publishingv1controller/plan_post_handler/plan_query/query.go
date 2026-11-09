package plan_query

import (
	"context"
	"fmt"
	"time"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/publishingwire"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Feeds interface {
	Posted(ctx context.Context, since time.Time) (calendar.Feed, error)
}

func New(feeds Feeds, calendar calendar.Calendar, clock cptime.Clock) Query {
	return Query{feeds: feeds, calendar: calendar, clock: clock}
}

// The slot the calendar gives a post now: nothing is held.
type Query struct {
	feeds    Feeds
	calendar calendar.Calendar
	clock    cptime.Clock
}

func (q Query) Plan(ctx context.Context, platforms []calendar.Platform, notBefore time.Time) (*publishingv1.PlanPostResponse, error) {
	now := q.clock.Now()
	feed, err := q.feeds.Posted(ctx, calendar.PostedSince(now))
	if err != nil {
		return nil, fmt.Errorf("failed to read what each platform holds: %w", err)
	}

	slot, found := q.calendar.Slot(platforms, feed, notBefore, now)
	if !found {
		return &publishingv1.PlanPostResponse{}, nil
	}
	return &publishingv1.PlanPostResponse{Slot: publishingwire.Slot(slot)}, nil
}
