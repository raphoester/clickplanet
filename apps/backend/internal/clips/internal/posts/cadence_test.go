package posts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var cadence = posts.Cadence{Review: 4 * time.Hour, Spacing: 2 * time.Hour}

func times(planned []posts.Post) map[string]time.Time {
	at := make(map[string]time.Time, len(planned))
	for _, post := range planned {
		snapshot := post.TakeSnapshot()
		at[snapshot.Title] = snapshot.At
	}
	return at
}

func TestTheBestClipGoesOutAReviewFromNowAndEachNextOneASpacingLater(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 40, 0, 0, time.UTC)

	planned := cadence.Posts([]posts.Clip{clip(t, 2, "SECOND"), clip(t, 1, "BEST"), clip(t, 3, "THIRD")}, now, posts.LoadedDice(0))

	assert.Equal(t, map[string]time.Time{
		"BEST":   time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC),
		"SECOND": time.Date(2026, 10, 12, 18, 40, 0, 0, time.UTC),
		"THIRD":  time.Date(2026, 10, 12, 20, 40, 0, 0, time.UTC),
	}, times(planned))
	assert.Equal(t, "BEST", planned[0].TakeSnapshot().Title)
}

func TestAPostGoesOutOnAWholeMinuteNeverSoonerThanTheReview(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 40, 27, 500, time.UTC)

	planned := cadence.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))

	assert.Equal(t, time.Date(2026, 10, 12, 16, 41, 0, 0, time.UTC), planned[0].TakeSnapshot().At)
}

func TestNoClipIsNoPost(t *testing.T) {
	assert.Empty(t, cadence.Posts(nil, time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC), posts.LoadedDice(0)))
}

func TestAPostIsReviewableWhileItsTimeIsAnHourAway(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	post := cadence.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))[0]

	assert.True(t, post.Reviewable(now))
	assert.True(t, post.Reviewable(now.Add(3*time.Hour)))
	assert.False(t, post.Reviewable(now.Add(3*time.Hour+time.Second)))
	assert.False(t, post.Reviewable(now.Add(5*time.Hour)))
}
