package posts_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

// count postings an hour apart from the first, on the platforms.
func postings(count int, first time.Time, platforms ...posts.Platform) posts.Feed {
	feed := make(posts.Feed, 0, count)
	for i := range count {
		feed = append(feed, posts.Posting{At: first.Add(time.Duration(i) * time.Hour), Platforms: platforms})
	}
	return feed
}

var (
	both   = []posts.Platform{posts.TikTok, posts.YouTube}
	tiktok = []posts.Platform{posts.TikTok}
)

func TestARunMakesAsManyClipsAsThePlatformWithTheMostRoomTakesAndNoMoreThanPerRun(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)

	assert.Equal(t, 2, budget.Clips(2, both, nil, now))
	assert.Equal(t, 2, budget.Clips(2, both, postings(8, now.Add(-20*time.Hour), posts.TikTok, posts.YouTube), now))
	assert.Equal(t, 1, budget.Clips(2, both, postings(9, now.Add(-20*time.Hour), posts.TikTok, posts.YouTube), now))
	assert.Equal(t, 0, budget.Clips(2, both, postings(10, now.Add(-20*time.Hour), posts.TikTok, posts.YouTube), now))
}

func TestARunOnTikTokAloneCountsTikTokAlone(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)
	full := postings(10, now.Add(-20*time.Hour), posts.TikTok)

	assert.Equal(t, 0, budget.Clips(2, tiktok, full, now))
	assert.Equal(t, 2, budget.Clips(2, []posts.Platform{posts.YouTube}, full, now))
}

func TestABudgetIsCountedOverATrailingDayAndCountsThePostsStillToCome(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)
	// One a day old to the minute, four in the day, five still to go out.
	feed := append(postings(5, now.Add(-24*time.Hour), posts.TikTok), postings(5, now.Add(time.Minute), posts.TikTok)...)

	assert.Equal(t, 0, budget.Clips(2, tiktok, feed, now))
	assert.Equal(t, 1, budget.Clips(2, tiktok, feed, now.Add(time.Minute)))
}

func TestAPlatformTheBudgetDoesNotNameTakesNothing(t *testing.T) {
	assert.Equal(t, 0, posts.Budget{}.Clips(2, both, nil, time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)))
}

func TestTheFeedKeepsAPostOnThePlatformsThatTookIt(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)
	post := asap.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))[0]

	feed := posts.Feed{}.With(post, posts.Scheduled{ID: "post-1", Refused: map[posts.Platform]error{posts.YouTube: errors.New("no account")}})

	assert.Equal(t, posts.Feed{{At: now.Add(5 * time.Minute), Platforms: tiktok}}, feed)
	assert.Empty(t, posts.Feed{}.With(post, posts.Scheduled{}))
}

func TestAPlatformThatTakesTheBestOnlyMakesOneClipARun(t *testing.T) {
	now := time.Date(2026, 10, 27, 14, 0, 0, 0, time.UTC)
	best := posts.Budget{posts.TikTok: {PerDay: 3, BestOnly: true}, posts.YouTube: {PerDay: 8}}

	assert.Equal(t, 1, best.Clips(2, tiktok, nil, now))
	assert.Equal(t, 2, best.Clips(2, both, nil, now))
	assert.Equal(t, 0, best.Clips(2, tiktok, postings(3, now.Add(-20*time.Hour), posts.TikTok), now))
}
