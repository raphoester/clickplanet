package remembered_feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/remembered_feed"
)

var (
	now     = time.Date(2026, 11, 12, 14, 52, 0, 0, time.UTC)
	youtube = []calendar.Platform{calendar.YouTube}
)

type listed struct {
	feed calendar.Feed
}

func (l *listed) Posted(context.Context, time.Time) (calendar.Feed, error) {
	return l.feed, nil
}

func TestAPostMadeHereCountsUntilTheListShowsIt(t *testing.T) {
	service := &listed{feed: calendar.Feed{{ID: "p-0", At: now.Add(-time.Hour), Platforms: youtube}}}
	feed := remembered_feed.New(service)
	made := calendar.Posting{ID: "p-1", At: now.Add(5 * time.Minute), Platforms: youtube}

	feed.Remember(made)
	before, err := feed.Posted(t.Context(), calendar.PostedSince(now))
	require.NoError(t, err)
	service.feed = append(service.feed, made)
	after, err := feed.Posted(t.Context(), calendar.PostedSince(now))
	require.NoError(t, err)

	assert.Equal(t, calendar.Feed{service.feed[0], made}, before, "not listed yet: remembered")
	assert.Equal(t, service.feed, after, "listed: counted once")
}

func TestAPostMadeADayAgoIsForgotten(t *testing.T) {
	feed := remembered_feed.New(&listed{})
	feed.Remember(calendar.Posting{ID: "p-1", At: now.Add(-25 * time.Hour), Platforms: youtube})

	posted, err := feed.Posted(t.Context(), calendar.PostedSince(now))

	require.NoError(t, err)
	assert.Empty(t, posted)
}

func TestAPostWithNoIDOrNoPlatformIsNotRemembered(t *testing.T) {
	feed := remembered_feed.New(&listed{})
	feed.Remember(calendar.Posting{At: now, Platforms: youtube})
	feed.Remember(calendar.Posting{ID: "p-1", At: now})

	posted, err := feed.Posted(t.Context(), calendar.PostedSince(now))

	require.NoError(t, err)
	assert.Empty(t, posted)
}
