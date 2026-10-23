package posts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func rendered(pick posts.Pick, title string) posts.ClipSnapshot {
	return posts.ClipSnapshot{
		Pick: pick, Title: title, Teasers: []string{"ALGERIA TAKES " + title, "ALGERIA INVADES " + title},
		Countries: []posts.TaggedCountry{{Code: "dz", Tag: "algeria"}}, ContinentTag: "africa",
		Key: posts.StoryKey{Flag: "dz", Continent: title}, Video: posts.VideoFile("clip-" + title + ".mp4"),
		TeaserNames: []string{"takes", "invades"}, Record: posts.ClipRecord(`{"format":"globe-replay","generator":"8dc31d2"}`),
	}
}

func clip(t *testing.T, pick posts.Pick, title string) posts.Clip {
	t.Helper()

	restored, err := rendered(pick, title).Restore()
	require.NoError(t, err)
	return restored
}

func TestAClipIsWhatTheGeneratorWrote(t *testing.T) {
	written := rendered(1, "FRANCE IS ATTACKING EUROPE")

	restored, err := written.Restore()

	require.NoError(t, err)
	assert.Equal(t, written, restored.TakeSnapshot())
}

func TestAClipWithAPartMissingIsRefused(t *testing.T) {
	for name, broken := range map[string]func(*posts.ClipSnapshot){
		"no pick":               func(s *posts.ClipSnapshot) { s.Pick = 0 },
		"no title":              func(s *posts.ClipSnapshot) { s.Title = "" },
		"no teaser":             func(s *posts.ClipSnapshot) { s.Teasers = nil },
		"a blank one":           func(s *posts.ClipSnapshot) { s.Teasers = []string{"ALGERIA TAKES AFRICA", ""} },
		"no tag":                func(s *posts.ClipSnapshot) { s.Countries = []posts.TaggedCountry{{Code: "dz"}} },
		"no flag":               func(s *posts.ClipSnapshot) { s.Key.Flag = "" },
		"no place":              func(s *posts.ClipSnapshot) { s.Key.Continent = "" },
		"no video":              func(s *posts.ClipSnapshot) { s.Video = "" },
		"a teaser with no name": func(s *posts.ClipSnapshot) { s.TeaserNames = []string{"takes", ""} },
		"a name short":          func(s *posts.ClipSnapshot) { s.TeaserNames = []string{"takes"} },
	} {
		t.Run(name, func(t *testing.T) {
			written := rendered(1, "FRANCE IS ATTACKING EUROPE")
			broken(&written)

			_, err := written.Restore()

			require.ErrorIs(t, err, posts.ErrInvalidClip)
		})
	}
}
