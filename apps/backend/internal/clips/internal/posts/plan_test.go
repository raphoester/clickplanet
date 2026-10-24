package posts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var (
	budget = posts.Budget{posts.TikTok: 10, posts.YouTube: 8}
	plan   = posts.Plan{Review: 4 * time.Hour, Spacing: 2 * time.Hour, Platforms: []posts.Platform{posts.TikTok, posts.YouTube}, Budget: budget}
	// As the job runs: no review, an hour between two posts on a platform.
	asap = posts.Plan{Spacing: time.Hour, Platforms: []posts.Platform{posts.TikTok, posts.YouTube}, Budget: budget}
)

func times(planned []posts.Post) map[string]time.Time {
	at := make(map[string]time.Time, len(planned))
	for _, post := range planned {
		snapshot := post.TakeSnapshot()
		at[snapshot.Title] = snapshot.At
	}
	return at
}

func next(t *testing.T, p posts.Plan, feed posts.Feed, at time.Time) (posts.PostSnapshot, bool) {
	t.Helper()

	post, room := p.Next(clip(t, 1, "BEST"), 1, 1, feed, at, posts.LoadedDice(0))
	return post.TakeSnapshot(), room
}

func TestTheBestClipGoesOutAReviewFromNowAndEachNextOneASpacingLater(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 40, 0, 0, time.UTC)

	planned := plan.Posts([]posts.Clip{clip(t, 2, "SECOND"), clip(t, 1, "BEST"), clip(t, 3, "THIRD")}, now, posts.LoadedDice(0))

	assert.Equal(t, map[string]time.Time{
		"BEST":   time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC),
		"SECOND": time.Date(2026, 10, 12, 18, 40, 0, 0, time.UTC),
		"THIRD":  time.Date(2026, 10, 12, 20, 40, 0, 0, time.UTC),
	}, times(planned))
	assert.Equal(t, "BEST", planned[0].TakeSnapshot().Title)
}

func TestWithNoReviewAPostGoesOutTheLeadFromNowOnAWholeMinute(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 53, 27, 500, time.UTC)

	snapshot, room := next(t, asap, nil, now)

	assert.True(t, room)
	assert.Equal(t, time.Date(2026, 10, 24, 12, 59, 0, 0, time.UTC), snapshot.At)
}

func TestAPostGoesOutOnAWholeMinuteNeverSoonerThanTheReview(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 40, 27, 500, time.UTC)

	planned := plan.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))

	assert.Equal(t, time.Date(2026, 10, 12, 16, 41, 0, 0, time.UTC), planned[0].TakeSnapshot().At)
}

func TestARunsSecondClipGoesOutASpacingAfterItsFirst(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 53, 0, 0, time.UTC)

	planned := asap.Posts([]posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}, now, posts.LoadedDice(0))

	assert.Equal(t, map[string]time.Time{
		"BEST":   time.Date(2026, 10, 24, 12, 58, 0, 0, time.UTC),
		"SECOND": time.Date(2026, 10, 24, 13, 58, 0, 0, time.UTC),
	}, times(planned))
}

func TestAPostKeepsASpacingAfterTheLastPostOnItsPlatformsWhoeverMadeIt(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 53, 0, 0, time.UTC)
	feed := posts.Feed{
		{At: now.Add(-3 * time.Hour), Platforms: []posts.Platform{posts.TikTok, posts.YouTube}},
		// The second clip of the run before, still to go out on TikTok.
		{At: now.Add(20 * time.Minute), Platforms: []posts.Platform{posts.TikTok}},
	}

	snapshot, _ := next(t, asap, feed, now)

	assert.Equal(t, now.Add(80*time.Minute), snapshot.At)
}

func TestAPostSoonAfterTheLastOneWaitsOnlyForTheSpacing(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 53, 0, 0, time.UTC)
	feed := posts.Feed{{At: now.Add(-50 * time.Minute), Platforms: []posts.Platform{posts.TikTok, posts.YouTube}}}

	snapshot, _ := next(t, asap, feed, now)

	assert.Equal(t, now.Add(10*time.Minute), snapshot.At)
}

func TestAPlatformWhoseDayIsFullIsLeftOutOfThePost(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 53, 0, 0, time.UTC)
	feed := postings(8, now.Add(-23*time.Hour), posts.TikTok, posts.YouTube)

	snapshot, room := next(t, asap, feed, now)

	assert.True(t, room)
	assert.Equal(t, []posts.Platform{posts.TikTok}, snapshot.Platforms)
}

func TestAClipWithNoPlatformLeftIsNoPost(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 53, 0, 0, time.UTC)
	feed := append(postings(8, now.Add(-23*time.Hour), posts.TikTok, posts.YouTube), postings(2, now.Add(-2*time.Hour), posts.TikTok)...)

	_, room := next(t, asap, feed, now)

	assert.False(t, room)
}

func TestARunTakesNoMoreThanAPlatformHasLeft(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 53, 0, 0, time.UTC)
	tight := posts.Plan{Spacing: time.Hour, Platforms: []posts.Platform{posts.TikTok}, Budget: posts.Budget{posts.TikTok: 1}}

	planned := tight.Posts([]posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}, now, posts.LoadedDice(0))

	assert.Len(t, planned, 1)
	assert.Equal(t, "BEST", planned[0].TakeSnapshot().Title)
}

func TestNoClipIsNoPost(t *testing.T) {
	assert.Empty(t, plan.Posts(nil, time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC), posts.LoadedDice(0)))
}

func TestEachPostGoesToEveryPlatformOfThePlanWithRoomLeft(t *testing.T) {
	planned := plan.Posts([]posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}, now, posts.LoadedDice(0))

	for _, post := range planned {
		assert.Equal(t, []posts.Platform{posts.TikTok, posts.YouTube}, post.TakeSnapshot().Platforms)
	}
}

func TestAPostSaysItsPlaceInItsRun(t *testing.T) {
	planned := asap.Posts([]posts.Clip{clip(t, 2, "SECOND"), clip(t, 1, "BEST")}, now, posts.LoadedDice(0))

	assert.Equal(t, [][2]int{{1, 2}, {2, 2}}, [][2]int{
		{planned[0].TakeSnapshot().RunPosition, planned[0].TakeSnapshot().RunSize},
		{planned[1].TakeSnapshot().RunPosition, planned[1].TakeSnapshot().RunSize},
	})
}
