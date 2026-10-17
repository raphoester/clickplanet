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
}

func (h *history) Told(_ context.Context, since time.Time) ([]posts.Telling, error) {
	h.since = append(h.since, since)
	return h.told, h.err
}

func TestARunTakesTheBestStoriesNotToldLately(t *testing.T) {
	past := &history{told: []posts.Telling{{Key: algeriaInAfrica, At: now.Add(-4 * time.Hour)}}}

	choice, err := choose_clips_usecase.New(written{shortlist: shortlistOf(t)}, past, cptime.NewFixedClock(now), 2).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{3, 4}, choice.Chosen.Picks())
	assert.Equal(t, []posts.Pick{1, 2}, choice.Retold.Picks())
	assert.Equal(t, []time.Time{now.Add(-7 * 24 * time.Hour)}, past.since)
}

func TestARunWithNoStoryWorthAClipTakesNone(t *testing.T) {
	choice, err := choose_clips_usecase.New(written{}, &history{}, cptime.NewFixedClock(now), 2).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, choice.Chosen)
}

func TestAShortlistThatCannotBeReadFailsTheRun(t *testing.T) {
	unreadable := errors.New("the file is gone")

	_, err := choose_clips_usecase.New(written{err: unreadable}, &history{}, cptime.NewFixedClock(now), 2).Execute(t.Context())

	require.ErrorIs(t, err, unreadable)
}

func TestAHistoryThatCannotBeReadFailsTheRunRatherThanTellAStoryTwice(t *testing.T) {
	unreachable := errors.New("zernio could not be reached")

	_, err := choose_clips_usecase.New(written{shortlist: shortlistOf(t)}, &history{err: unreachable}, cptime.NewFixedClock(now), 2).Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
}
