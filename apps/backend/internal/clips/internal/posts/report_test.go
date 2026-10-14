package posts_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func TestAReportSaysWhatBecameOfEachClip(t *testing.T) {
	planned := cadence.Posts([]posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}, now, posts.LoadedDice(0))

	report := posts.Report{}.
		With(planned[0], "post-1", nil).
		With(planned[1], "", errors.New("the caption is too long"))

	assert.Equal(t, posts.Report{
		{Title: "BEST", At: planned[0].TakeSnapshot().At, Post: "post-1"},
		{Title: "SECOND", At: planned[1].TakeSnapshot().At, Failure: "the caption is too long"},
	}, report)
	assert.False(t, report.Complete())
}

func TestARunThatScheduledEveryClipIsComplete(t *testing.T) {
	planned := cadence.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))

	assert.True(t, posts.Report{}.With(planned[0], "post-1", nil).Complete())
}

func TestAddingToAReportLeavesItAsItWas(t *testing.T) {
	planned := cadence.Posts([]posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}, now, posts.LoadedDice(0))
	first := posts.Report{}.With(planned[0], "post-1", nil)

	_ = first.With(planned[1], "post-2", nil)

	assert.Len(t, first, 1)
}
