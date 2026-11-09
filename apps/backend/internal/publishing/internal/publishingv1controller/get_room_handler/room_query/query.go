package room_query

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

// What the service holds now, read once, against the calendar.
type Query struct {
	feeds    Feeds
	calendar calendar.Calendar
	clock    cptime.Clock
}

func (q Query) Room(ctx context.Context, platforms []calendar.Platform) ([]*publishingv1.PlatformRoom, error) {
	now := q.clock.Now()
	feed, err := q.feeds.Posted(ctx, calendar.PostedSince(now))
	if err != nil {
		return nil, fmt.Errorf("failed to read what each platform holds: %w", err)
	}

	rooms := make([]*publishingv1.PlatformRoom, 0, len(platforms))
	for _, platform := range platforms {
		rooms = append(rooms, &publishingv1.PlatformRoom{
			Platform: publishingwire.Platform(platform),
			Left:     uint32(q.calendar.Left(platform, feed, now)), //nolint:gosec // a day's posts, a dozen.
		})
	}
	return rooms, nil
}
