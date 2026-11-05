package keep_appearances_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/keep_appearances_usecase"
)

type server struct {
	kept    []posts.Appearance
	refuses posts.PostID
}

func (s *server) Keep(_ context.Context, appearance posts.Appearance) (int, error) {
	if appearance.Post == s.refuses {
		return 0, errors.New("no cast of that id")
	}
	s.kept = append(s.kept, appearance)
	return len(appearance.Fighters), nil
}

func outcome(post posts.PostID, fighters ...posts.FighterNumber) posts.Outcome {
	return posts.Outcome{
		Title: "FRANCE VS TURKEY", At: time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC), Post: post, Cast: "c", Fighters: fighters,
		Platforms: []posts.PlatformOutcome{{Platform: posts.TikTok}},
	}
}

func TestEveryPostThatNamesSomeoneIsKeptAndOneRefusedStopsNoOther(t *testing.T) {
	backend := &server{refuses: "p1"}

	kept := keep_appearances_usecase.New(backend).Execute(t.Context(), posts.Report{outcome("p1", 1), outcome("p2"), outcome("p3", 2, 3)})

	require.Len(t, kept, 2)
	require.Error(t, kept[0].Err)
	require.NoError(t, kept[1].Err)
	assert.Equal(t, 2, kept[1].Named)
	require.Len(t, backend.kept, 1)
	assert.Equal(t, posts.PostID("p3"), backend.kept[0].Post)
}
