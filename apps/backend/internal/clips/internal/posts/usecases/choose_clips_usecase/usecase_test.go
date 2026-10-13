package choose_clips_usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
)

type written struct {
	shortlist posts.Shortlist
	err       error
}

func (w written) Shortlist(context.Context) (posts.Shortlist, error) {
	return w.shortlist, w.err
}

func TestARunTakesTheBestStoriesOfTheShortlist(t *testing.T) {
	shortlist := posts.ShortlistOf([]string{"BEST", "SECOND", "THIRD"})

	chosen, err := choose_clips_usecase.New(written{shortlist: shortlist}, 2).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{1, 2}, chosen.Picks())
}

func TestARunWithNoStoryWorthAClipTakesNone(t *testing.T) {
	chosen, err := choose_clips_usecase.New(written{}, 2).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, chosen)
}

func TestAShortlistThatCannotBeReadFailsTheRun(t *testing.T) {
	unreadable := errors.New("the file is gone")

	_, err := choose_clips_usecase.New(written{err: unreadable}, 2).Execute(t.Context())

	require.ErrorIs(t, err, unreadable)
}
