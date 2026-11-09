package posts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
)

var at = time.Date(2026, 11, 12, 14, 57, 0, 0, time.UTC)

func draft() posts.Draft {
	return posts.Draft{
		Slot:  calendar.Slot{At: at, Platforms: []calendar.Platform{calendar.TikTok, calendar.YouTube}},
		Media: "https://media.zernio.com/temp/clip-1.mp4", Key: "f00d", Title: "ITALY RETAKES ITS LAND FROM PORTUGAL",
		TikTok:   &posts.TikTokWords{Caption: "ITALY RETAKES ITS LAND FROM PORTUGAL | CLICKPLANET"},
		YouTube:  &posts.YouTubeWords{Title: "ITALY RETAKES ITS LAND | CLICKPLANET", Description: "Play", Tags: []string{"map"}},
		Metadata: map[string]any{"storyFlag": "it"},
	}
}

func TestAPostIsWhatTheJobAsked(t *testing.T) {
	post, err := posts.NewPost(draft())
	require.NoError(t, err)

	assert.Equal(t, posts.PostSnapshot{
		At: at, Platforms: []calendar.Platform{calendar.TikTok, calendar.YouTube},
		Media: "https://media.zernio.com/temp/clip-1.mp4", Key: "f00d", Title: "ITALY RETAKES ITS LAND FROM PORTUGAL",
		TikTok:   posts.TikTokWords{Caption: "ITALY RETAKES ITS LAND FROM PORTUGAL | CLICKPLANET"},
		YouTube:  posts.YouTubeWords{Title: "ITALY RETAKES ITS LAND | CLICKPLANET", Description: "Play", Tags: []string{"map"}},
		Metadata: map[string]any{"storyFlag": "it"},
	}, post.TakeSnapshot())
	assert.Equal(t, calendar.Slot{At: at, Platforms: []calendar.Platform{calendar.TikTok, calendar.YouTube}}, post.Slot())
}

func TestAPostNeedsOnlyTheWordsOfItsOwnPlatforms(t *testing.T) {
	youtubeOnly := draft()
	youtubeOnly.Slot.Platforms = []calendar.Platform{calendar.YouTube}
	youtubeOnly.TikTok = nil

	_, err := posts.NewPost(youtubeOnly)

	require.NoError(t, err)
}

func TestAPostIsRefusedForEveryReasonAtOnce(t *testing.T) {
	bad := draft()
	bad.Slot = calendar.Slot{Platforms: []calendar.Platform{calendar.TikTok, calendar.YouTube, "instagram"}}
	bad.Media = "http://media.zernio.com/clip.mp4"
	bad.Key = ""
	bad.TikTok = &posts.TikTokWords{}
	bad.YouTube = nil

	_, err := posts.NewPost(bad)

	require.ErrorIs(t, err, posts.ErrInvalidPost)
	for _, reason := range []string{
		"no time", "no https address", "idempotency key", "tiktok with no caption", "youtube with no title", `"instagram"`,
	} {
		assert.ErrorContains(t, err, reason)
	}
	_, err = posts.NewPost(posts.Draft{})
	assert.ErrorContains(t, err, "names no platform")
}

func TestOnlyAnMP4WithANameAndASizeIsUploaded(t *testing.T) {
	require.NoError(t, posts.Media{FileName: "clip-1.mp4", ContentType: posts.VideoMP4, Size: 80 << 20}.FileError())

	err := posts.Media{FileName: "../clip.mp4", ContentType: "image/png", Size: 0}.FileError()

	require.ErrorIs(t, err, posts.ErrInvalidMedia)
	for _, reason := range []string{"a path", `"image/png"`, "0 bytes"} {
		assert.ErrorContains(t, err, reason)
	}
	assert.Error(t, posts.Media{FileName: "clip.mp4", ContentType: posts.VideoMP4, Size: 2 << 30}.FileError())
}
