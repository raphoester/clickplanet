package posts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var (
	both   = []posts.Platform{posts.TikTok, posts.YouTube}
	tiktok = []posts.Platform{posts.TikTok}

	plan = posts.Plan{Platforms: both}
	// As the job runs: TikTok for a run's best clip alone.
	asap = posts.Plan{Platforms: both, BestOnly: tiktok}
)

func TestTheBestClipGoesFirstWhateverTheOrderItWasRenderedIn(t *testing.T) {
	ranked := posts.Ranked([]posts.Clip{clip(t, 2, "SECOND"), clip(t, 1, "BEST"), clip(t, 3, "THIRD")})

	titles := make([]string, 0, len(ranked))
	for _, clip := range ranked {
		titles = append(titles, clip.TakeSnapshot().Title)
	}
	assert.Equal(t, []string{"BEST", "SECOND", "THIRD"}, titles)
}

func TestAPlatformThatTakesTheBestOnlyIsAskedForARunsFirstClipAlone(t *testing.T) {
	assert.Equal(t, both, asap.PlatformsOf(1))
	assert.Equal(t, []posts.Platform{posts.YouTube}, asap.PlatformsOf(2))
	assert.Equal(t, both, plan.PlatformsOf(2), "with none that takes the best alone, every clip asks for every platform")
}

func TestARunMakesAsManyClipsAsThePlatformWithTheMostRoomTakesAndNoMoreThanPerRun(t *testing.T) {
	assert.Equal(t, 2, plan.Clips(2, map[posts.Platform]int{posts.TikTok: 10, posts.YouTube: 8}))
	assert.Equal(t, 1, plan.Clips(2, map[posts.Platform]int{posts.TikTok: 0, posts.YouTube: 1}))
	assert.Equal(t, 0, plan.Clips(2, map[posts.Platform]int{posts.TikTok: 0, posts.YouTube: 0}))
	assert.Equal(t, 0, plan.Clips(2, nil))
}

func TestAPlatformThatTakesTheBestOnlyMakesOneClipARun(t *testing.T) {
	assert.Equal(t, 1, asap.Clips(2, map[posts.Platform]int{posts.TikTok: 3, posts.YouTube: 0}))
	assert.Equal(t, 2, asap.Clips(2, map[posts.Platform]int{posts.TikTok: 3, posts.YouTube: 12}))
	assert.Equal(t, 0, asap.Clips(2, map[posts.Platform]int{posts.TikTok: 0, posts.YouTube: 0}))
}

func TestARunOnTikTokAloneCountsTikTokAlone(t *testing.T) {
	alone := posts.Plan{Platforms: tiktok}

	assert.Equal(t, 0, alone.Clips(2, map[posts.Platform]int{posts.TikTok: 0, posts.YouTube: 12}))
	assert.Equal(t, 2, alone.Clips(2, map[posts.Platform]int{posts.TikTok: 3}))
}

func TestAPostIsItsClipOnThePlatformsOfItsPlaceUntilItsSlotSaysOtherwise(t *testing.T) {
	post := asap.Post(clip(t, 2, "SECOND"), 2, 2, posts.LoadedDice(0))

	assert.Equal(t, []posts.Platform{posts.YouTube}, post.TakeSnapshot().Platforms)
	assert.Equal(t, [2]int{2, 2}, [2]int{post.TakeSnapshot().RunPosition, post.TakeSnapshot().RunSize})

	at := time.Date(2026, 10, 24, 12, 58, 0, 0, time.UTC)
	placed := post.At(posts.Slot{At: at, Platforms: []posts.Platform{posts.YouTube}})
	assert.Equal(t, at, placed.TakeSnapshot().At)
	assert.True(t, post.TakeSnapshot().At.IsZero(), "a slot makes a new post")
}
