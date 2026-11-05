package posts_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func scheduledOn(failures ...string) []posts.PlatformOutcome {
	return []posts.PlatformOutcome{{Platform: posts.TikTok, Failure: failures[0]}, {Platform: posts.YouTube, Failure: failures[1]}}
}

func TestAReportSaysWhatBecameOfEachClipOnEachPlatform(t *testing.T) {
	planned := plan.Posts([]posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}, now, posts.LoadedDice(0))

	report := posts.Report{}.
		With(planned[0], posts.Scheduled{ID: "post-1"}, nil).
		With(planned[1], posts.Scheduled{}, errors.New("the caption is too long"))

	assert.Equal(t, posts.Report{
		{Title: "BEST", At: planned[0].TakeSnapshot().At, Post: "post-1", Platforms: scheduledOn("", "")},
		{
			Title: "SECOND", At: planned[1].TakeSnapshot().At,
			Platforms: scheduledOn("the caption is too long", "the caption is too long"),
		},
	}, report)
	assert.False(t, report.Complete())
}

func TestAPlatformThatRefusedAClipIsInTheReportBesideTheOneThatTookIt(t *testing.T) {
	planned := plan.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))

	report := posts.Report{}.With(planned[0], posts.Scheduled{
		ID: "post-1", Refused: map[posts.Platform]error{posts.YouTube: errors.New("no youtube account")},
	}, nil)

	assert.Equal(t, scheduledOn("", "no youtube account"), report[0].Platforms)
	assert.Equal(t, posts.PostID("post-1"), report[0].Post)
	assert.False(t, report.Complete())
}

func TestARunThatScheduledEveryClipOnEveryPlatformIsComplete(t *testing.T) {
	planned := plan.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))

	assert.True(t, posts.Report{}.With(planned[0], posts.Scheduled{ID: "post-1"}, nil).Complete())
}

func TestAClipThatIsNoPostIsNeverComplete(t *testing.T) {
	planned := plan.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))

	assert.False(t, posts.Report{}.With(planned[0], posts.Scheduled{}, nil).Complete())
}

func TestAClipHeldForAFullDayIsInTheReportAndIsNoFailure(t *testing.T) {
	planned := plan.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))

	report := posts.Report{}.With(planned[0], posts.Scheduled{ID: "post-1"}, nil).WithHeld(clip(t, 2, "SECOND"))

	assert.Equal(t, posts.Outcome{Title: "SECOND", Held: true}, report[1])
	assert.True(t, report.Complete())
}

func TestAddingToAReportLeavesItAsItWas(t *testing.T) {
	planned := plan.Posts([]posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}, now, posts.LoadedDice(0))
	first := posts.Report{}.With(planned[0], posts.Scheduled{ID: "post-1"}, nil)

	_ = first.With(planned[1], posts.Scheduled{ID: "post-2"}, nil)

	assert.Len(t, first, 1)
}

func TestOnlyAPostMadeThatNamesSomeoneIsAnAppearanceOnThePlatformsThatTookIt(t *testing.T) {
	at := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	report := posts.Report{
		{
			Title: "FRANCE VS TURKEY", At: at, Post: "p1", Cast: "c", Fighters: []posts.FighterNumber{2, 1},
			Platforms: []posts.PlatformOutcome{{Platform: posts.TikTok, Failure: "refused"}, {Platform: posts.YouTube}},
		},
		{Title: "NOBODY NAMED", At: at, Post: "p2", Platforms: []posts.PlatformOutcome{{Platform: posts.YouTube}}},
		{Title: "HELD", Held: true, Cast: "c", Fighters: []posts.FighterNumber{1}},
		{
			Title: "REFUSED EVERYWHERE", At: at, Post: "p3", Cast: "c", Fighters: []posts.FighterNumber{1},
			Platforms: []posts.PlatformOutcome{{Platform: posts.YouTube, Failure: "refused"}},
		},
		{
			Title: "NEVER MADE", At: at, Cast: "c", Fighters: []posts.FighterNumber{1},
			Platforms: []posts.PlatformOutcome{{Platform: posts.YouTube, Failure: "down"}},
		},
	}

	assert.Equal(t, []posts.Appearance{{
		Cast: "c", Fighters: []posts.FighterNumber{2, 1}, Post: "p1", Title: "FRANCE VS TURKEY", Platforms: []posts.Platform{posts.YouTube}, At: at,
	}}, report.Appearances())
}
