package posts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var (
	budget = posts.Budget{posts.TikTok: {PerDay: 10}, posts.YouTube: {PerDay: 8}}
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
	tight := posts.Plan{Spacing: time.Hour, Platforms: []posts.Platform{posts.TikTok}, Budget: posts.Budget{posts.TikTok: {PerDay: 1}}}

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

var paced = posts.Plan{Spacing: time.Hour, Platforms: []posts.Platform{posts.TikTok, posts.YouTube}, Budget: posts.Budget{
	posts.TikTok:  {PerDay: 3, Hours: posts.Hours{From: 14, To: 20}, BestOnly: true},
	posts.YouTube: {PerDay: 8},
}}

func TestAPlatformOutOfItsHoursIsLeftOutOfThePost(t *testing.T) {
	snapshot, room := next(t, paced, nil, time.Date(2026, 10, 27, 12, 53, 0, 0, time.UTC))

	assert.True(t, room)
	assert.Equal(t, []posts.Platform{posts.YouTube}, snapshot.Platforms)
}

func TestAPostInAPlatformsHoursGoesOutThere(t *testing.T) {
	snapshot, _ := next(t, paced, nil, time.Date(2026, 10, 27, 13, 55, 0, 0, time.UTC))

	assert.Equal(t, time.Date(2026, 10, 27, 14, 0, 0, 0, time.UTC), snapshot.At)
	assert.Equal(t, []posts.Platform{posts.TikTok, posts.YouTube}, snapshot.Platforms)
}

func TestAPlatformThatTakesTheBestOnlyIsLeftOutOfARunsNextClips(t *testing.T) {
	now := time.Date(2026, 10, 27, 14, 53, 0, 0, time.UTC)

	planned := paced.Posts([]posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}, now, posts.LoadedDice(0))

	assert.Equal(t, []posts.Platform{posts.TikTok, posts.YouTube}, planned[0].TakeSnapshot().Platforms)
	assert.Equal(t, []posts.Platform{posts.YouTube}, planned[1].TakeSnapshot().Platforms)
}

func TestAPlatformLeftOutForItsHoursNoLongerHoldsThePostBack(t *testing.T) {
	now := time.Date(2026, 10, 27, 19, 0, 0, 0, time.UTC)
	// A post by hand on TikTok: its spacing would push the clip past TikTok's hours.
	feed := posts.Feed{{At: now.Add(30 * time.Minute), Platforms: []posts.Platform{posts.TikTok}}}

	snapshot, _ := next(t, paced, feed, now)

	assert.Equal(t, []posts.Platform{posts.YouTube}, snapshot.Platforms)
	assert.Equal(t, now.Add(5*time.Minute), snapshot.At)
}

func TestAClipOutOfTheHoursOfItsOnlyPlatformIsNoPost(t *testing.T) {
	tiktok := paced
	tiktok.Platforms = []posts.Platform{posts.TikTok}

	_, room := next(t, tiktok, nil, time.Date(2026, 10, 27, 20, 0, 0, 0, time.UTC))

	assert.False(t, room)
}

// A post scheduled ahead, by hand or a battle's teaser, is a reservation: it holds back no post before it.
func TestAPostScheduledAheadHoldsBackNoPostBeforeIt(t *testing.T) {
	now := time.Date(2026, 11, 12, 14, 52, 0, 0, time.UTC)

	for _, ahead := range []time.Time{
		time.Date(2026, 11, 12, 18, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 13, 18, 0, 0, 0, time.UTC),
	} {
		feed := posts.Feed{{At: ahead, Platforms: both}}

		snapshot, room := next(t, paced, feed, now)

		assert.True(t, room)
		assert.Equal(t, time.Date(2026, 11, 12, 14, 57, 0, 0, time.UTC), snapshot.At, "with a post at %s", ahead)
		assert.Equal(t, both, snapshot.Platforms, "with a post at %s", ahead)
	}
}

func TestAPostGoesInTheFirstGapThatKeepsTheSpacingOnBothSides(t *testing.T) {
	now := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	// The second clip of the run before at 10:30, and a post by hand at 12:30: 11:30 is an hour from both.
	feed := posts.Feed{
		{At: now.Add(30 * time.Minute), Platforms: both},
		{At: now.Add(150 * time.Minute), Platforms: []posts.Platform{posts.YouTube}},
	}

	snapshot, _ := next(t, asap, feed, now)

	assert.Equal(t, now.Add(90*time.Minute), snapshot.At)
}

func TestAPostSkipsAGapTooShortForTheSpacingOnBothSides(t *testing.T) {
	now := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	// 11:30 is an hour after the post at 10:30 and half an hour before the one at 12:00: the post goes an hour after that.
	feed := posts.Feed{
		{At: now.Add(30 * time.Minute), Platforms: both},
		{At: now.Add(2 * time.Hour), Platforms: []posts.Platform{posts.YouTube}},
	}

	snapshot, _ := next(t, asap, feed, now)

	assert.Equal(t, now.Add(3*time.Hour), snapshot.At)
}

func TestAPostThatWouldOverfillTheDayOfAPostScheduledAheadLeavesItsPlatformOut(t *testing.T) {
	now := time.Date(2026, 11, 12, 18, 52, 0, 0, time.UTC)
	// Two today, and one tomorrow at 14:30: the trailing day that ends with it holds all three.
	feed := posts.Feed{
		{At: time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC), Platforms: both},
		{At: time.Date(2026, 11, 12, 17, 0, 0, 0, time.UTC), Platforms: both},
		{At: time.Date(2026, 11, 13, 14, 30, 0, 0, time.UTC), Platforms: tiktok},
	}

	snapshot, room := next(t, paced, feed, now)

	assert.True(t, room)
	assert.Equal(t, []posts.Platform{posts.YouTube}, snapshot.Platforms)
	assert.Equal(t, now.Add(5*time.Minute), snapshot.At)
}

func TestAPostScheduledAheadCountsOnlyInTheTrailingDaysThatHoldIt(t *testing.T) {
	now := time.Date(2026, 11, 12, 18, 52, 0, 0, time.UTC)
	// Two today, and one tomorrow at 15:30: the day that ends with it starts after today's first.
	feed := posts.Feed{
		{At: time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC), Platforms: both},
		{At: time.Date(2026, 11, 12, 17, 0, 0, 0, time.UTC), Platforms: both},
		{At: time.Date(2026, 11, 13, 15, 30, 0, 0, time.UTC), Platforms: tiktok},
	}

	snapshot, _ := next(t, paced, feed, now)

	assert.Equal(t, both, snapshot.Platforms)
}
