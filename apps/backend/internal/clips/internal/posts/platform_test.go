package posts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func TestThePlatformsAreTheOnesNamedInTheirOrder(t *testing.T) {
	platforms, err := posts.PlatformsOf("youtube", "tiktok")

	require.NoError(t, err)
	assert.Equal(t, []posts.Platform{posts.YouTube, posts.TikTok}, platforms)
}

func TestPlatformsThatAreNotEachKnownOnceAreRefused(t *testing.T) {
	for name, names := range map[string][]string{
		"none":           {},
		"an unknown one": {"instagram"},
		"in capitals":    {"TikTok"},
		"one twice":      {"tiktok", "youtube", "tiktok"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := posts.PlatformsOf(names...)

			require.ErrorIs(t, err, posts.ErrInvalidPlatforms)
		})
	}
}
