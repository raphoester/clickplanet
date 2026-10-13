package posts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func rendered(pick posts.Pick, title string) posts.ClipSnapshot {
	return posts.ClipSnapshot{
		Pick: pick, Title: title, Caption: title + ". Who stops them? 👇\nclickplanet.lol\n#clickplanet",
		Video: posts.VideoFile("clip-" + title + ".mp4"),
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
		"no pick":    func(s *posts.ClipSnapshot) { s.Pick = 0 },
		"no title":   func(s *posts.ClipSnapshot) { s.Title = "" },
		"no caption": func(s *posts.ClipSnapshot) { s.Caption = "" },
		"no video":   func(s *posts.ClipSnapshot) { s.Video = "" },
	} {
		t.Run(name, func(t *testing.T) {
			written := rendered(1, "FRANCE IS ATTACKING EUROPE")
			broken(&written)

			_, err := written.Restore()

			require.ErrorIs(t, err, posts.ErrInvalidClip)
		})
	}
}
