package appearances_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances"
)

func TestOnlyThePlatformsTheClipsGoToAreKnown(t *testing.T) {
	for _, name := range []string{"tiktok", "youtube"} {
		platform, err := appearances.PlatformOf(name)
		require.NoError(t, err)
		assert.Equal(t, name, string(platform))
	}
	_, err := appearances.PlatformOf("instagram")
	assert.ErrorIs(t, err, appearances.ErrUnknownPlatform)
}

func TestAPostOnNoPlatformNamesNoOne(t *testing.T) {
	_, err := appearances.PostOf("p", "FRANCE VS TURKEY", nil, time.Now())
	assert.ErrorIs(t, err, appearances.ErrNowhere)
}

func TestAPostNeedsAnID(t *testing.T) {
	_, err := appearances.PostIDOf("")
	assert.ErrorIs(t, err, appearances.ErrNoPost)
}
