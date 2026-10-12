package posts_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func TestAReportSaysWhatBecameOfEachClip(t *testing.T) {
	planned := cadence.Posts([]posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}, now)

	report := posts.Report{}.
		With(planned[0], "post-1", nil).
		With(planned[1], "", errors.New("the caption is too long"))

	assert.Equal(t, posts.Report{
		{Title: "BEST", At: planned[0].TakeSnapshot().At, Post: "post-1"},
		{Title: "SECOND", At: planned[1].TakeSnapshot().At, Failure: "the caption is too long"},
	}, report)
	require.ErrorIs(t, report.UnsentError(), posts.ErrUnsent)
	assert.ErrorContains(t, report.UnsentError(), "1 of 2")
}

func TestARunThatScheduledEveryClipHasNothingUnsent(t *testing.T) {
	planned := cadence.Posts([]posts.Clip{clip(t, 1, "BEST")}, now)

	require.NoError(t, posts.Report{}.With(planned[0], "post-1", nil).UnsentError())
}

func TestAddingToAReportLeavesItAsItWas(t *testing.T) {
	planned := cadence.Posts([]posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}, now)
	first := posts.Report{}.With(planned[0], "post-1", nil)

	_ = first.With(planned[1], "post-2", nil)

	assert.Len(t, first, 1)
}
