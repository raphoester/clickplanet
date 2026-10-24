package choose_clips_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now             = time.Date(2026, 10, 14, 0, 57, 0, 0, time.UTC)
	algeriaInAfrica = posts.StoryKey{Flag: "dz", Continent: "Africa"}
	stories         = []posts.Story{
		{Title: "ALGERIA STRIKES BACK", Key: algeriaInAfrica},
		{Title: "ALGERIA VS FRANCE", Key: algeriaInAfrica},
		{Title: "FRANCE IS ATTACKING EUROPE", Key: posts.StoryKey{Flag: "fr", Continent: "Europe"}},
		{Title: "PORTUGAL STRIKES BACK", Key: posts.StoryKey{Flag: "pt", Continent: "Europe"}},
	}
)

func shortlistOf(t *testing.T) posts.Shortlist {
	t.Helper()

	shortlist, err := posts.ShortlistOf(stories...)
	require.NoError(t, err)
	return shortlist
}

type written struct {
	shortlist posts.Shortlist
	err       error
}

func (w written) Shortlist(context.Context) (posts.Shortlist, error) {
	return w.shortlist, w.err
}

type history struct {
	told  []posts.Telling
	err   error
	since []time.Time
	// What each platform holds, and since when it was asked.
	feed       posts.Feed
	feedErr    error
	postedFrom []time.Time
}

func (h *history) Told(_ context.Context, since time.Time) ([]posts.Telling, error) {
	h.since = append(h.since, since)
	return h.told, h.err
}

func (h *history) Posted(_ context.Context, since time.Time) (posts.Feed, error) {
	h.postedFrom = append(h.postedFrom, since)
	return h.feed, h.feedErr
}

var room = choose_clips_usecase.Room{
	PerRun: 2, Budget: posts.Budget{posts.TikTok: 10, posts.YouTube: 8}, Platforms: []posts.Platform{posts.TikTok, posts.YouTube},
}

func choosing(w written, h *history) *choose_clips_usecase.UseCase {
	return choose_clips_usecase.New(w, h, h, cptime.NewFixedClock(now), room)
}

func TestARunTakesTheBestStoriesNotToldLately(t *testing.T) {
	past := &history{told: []posts.Telling{{Key: algeriaInAfrica, At: now.Add(-4 * time.Hour)}}}

	choice, err := choosing(written{shortlist: shortlistOf(t)}, past).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{3, 4}, choice.Chosen.Picks())
	assert.Equal(t, []posts.Pick{1, 2}, choice.Retold.Picks())
	assert.Equal(t, []time.Time{now.Add(-7 * 24 * time.Hour)}, past.since)
}

func TestARunWithNoStoryWorthAClipTakesNone(t *testing.T) {
	choice, err := choosing(written{}, &history{}).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, choice.Chosen)
}

func TestAShortlistThatCannotBeReadFailsTheRun(t *testing.T) {
	unreadable := errors.New("the file is gone")

	_, err := choosing(written{err: unreadable}, &history{}).Execute(t.Context())

	require.ErrorIs(t, err, unreadable)
}

func TestAHistoryThatCannotBeReadFailsTheRunRatherThanTellAStoryTwice(t *testing.T) {
	unreachable := errors.New("zernio could not be reached")

	_, err := choosing(written{shortlist: shortlistOf(t)}, &history{err: unreachable}).Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
}

// count posts on every platform, an hour apart, the newest an hour ago.
func hourly(count int) posts.Feed {
	feed := make(posts.Feed, 0, count)
	for i := range count {
		feed = append(feed, posts.Posting{At: now.Add(-time.Duration(i+1) * time.Hour), Platforms: room.Platforms})
	}
	return feed
}

func TestARunTakesNoMoreStoriesThanThePlatformsHaveRoomFor(t *testing.T) {
	past := &history{feed: hourly(9)}

	choice, err := choosing(written{shortlist: shortlistOf(t)}, past).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{1}, choice.Chosen.Picks())
	assert.Equal(t, 1, choice.Room)
	assert.Equal(t, []time.Time{now.Add(-24 * time.Hour)}, past.postedFrom)
}

func TestARunWhoseEveryPlatformPostedItsDayTakesNone(t *testing.T) {
	choice, err := choosing(written{shortlist: shortlistOf(t)}, &history{feed: hourly(10)}).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, choice.Chosen)
	assert.Equal(t, 0, choice.Room)
}

func TestAFeedThatCannotBeReadFailsTheRunRatherThanPostOverTheCap(t *testing.T) {
	unreachable := errors.New("zernio could not be reached")

	_, err := choosing(written{shortlist: shortlistOf(t)}, &history{feedErr: unreachable}).Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
}
