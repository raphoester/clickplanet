package posts_test

import (
	"fmt"
	"strings"
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

func TestATikTokCarouselIsUpTo35PhotosUnderATitle(t *testing.T) {
	carousel := draft()
	carousel.TikTokPhotos = []posts.MediaURL{"https://media.zernio.com/temp/card-1.jpg", "https://media.zernio.com/temp/card-2.jpg"}
	carousel.TikTok.Title = "RING WAR · SATURDAY 20:00 UTC"

	post, err := posts.NewPost(carousel)

	require.NoError(t, err)
	assert.Equal(t, carousel.TikTokPhotos, post.TakeSnapshot().TikTokPhotos)
	assert.Equal(t, "RING WAR · SATURDAY 20:00 UTC", post.TakeSnapshot().TikTok.Title)
}

func TestACarouselTikTokCannotPostIsRefusedForEveryReasonAtOnce(t *testing.T) {
	carousel := draft()
	for i := range 36 {
		carousel.TikTokPhotos = append(carousel.TikTokPhotos, posts.MediaURL(fmt.Sprintf("https://media.zernio.com/temp/card-%d.jpg", i)))
	}
	carousel.TikTokPhotos[3] = "http://media.zernio.com/temp/card-3.jpg"
	carousel.TikTok.Title = strings.Repeat("A", 91)

	_, err := posts.NewPost(carousel)

	require.ErrorIs(t, err, posts.ErrInvalidPost)
	for _, reason := range []string{"36 photos", "a photo is no https address", "longer than 90"} {
		assert.ErrorContains(t, err, reason)
	}

	untitled := draft()
	untitled.TikTokPhotos = []posts.MediaURL{"https://media.zernio.com/temp/card-1.jpg"}
	_, err = posts.NewPost(untitled)
	require.ErrorContains(t, err, "the photos have no title")

	untitled.Slot.Platforms = []calendar.Platform{calendar.YouTube}
	_, err = posts.NewPost(untitled)
	require.NoError(t, err, "photos TikTok is not in the slot of are never posted, and need no title")
}

func TestOnlyAnMP4OrAJPEGWithANameAndASizeIsUploaded(t *testing.T) {
	require.NoError(t, posts.Media{FileName: "clip-1.mp4", ContentType: posts.VideoMP4, Size: 80 << 20}.FileError())
	require.NoError(t, posts.Media{FileName: "card-1.jpg", ContentType: posts.PhotoJPEG, Size: 400 << 10}.FileError())
	require.ErrorIs(t, posts.Media{FileName: "card-1.jpg", ContentType: posts.PhotoJPEG, Size: 21 << 20}.FileError(), posts.ErrInvalidMedia)

	err := posts.Media{FileName: "../clip.mp4", ContentType: "image/png", Size: 0}.FileError()

	require.ErrorIs(t, err, posts.ErrInvalidMedia)
	for _, reason := range []string{"a path", `"image/png"`} {
		assert.ErrorContains(t, err, reason)
	}
	assert.ErrorContains(t, posts.Media{FileName: "clip.mp4", ContentType: posts.VideoMP4, Size: 0}.FileError(), "0 bytes")
	assert.Error(t, posts.Media{FileName: "clip.mp4", ContentType: posts.VideoMP4, Size: 2 << 30}.FileError())
}
