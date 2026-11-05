package videos_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
)

var (
	rules = videos.Rules{MaxBytes: 1000}
	at    = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	mp4   = videos.File{ID: videos.VideoID{1}, MediaType: "video/mp4", Size: 1000}
)

func TestAVideoIsAnMp4AMovOrAWebmOfAtMostTheMaxBytes(t *testing.T) {
	for _, media := range []videos.MediaType{"video/mp4", "video/quicktime", "video/webm"} {
		require.NoError(t, rules.FileError(videos.File{MediaType: media, Size: 1}), media)
	}
	for _, file := range []videos.File{
		{MediaType: "image/png", Size: 10},
		{MediaType: "text/html", Size: 10},
		{MediaType: "video/mp4", Size: 1001},
		{MediaType: "video/mp4", Size: 0},
	} {
		require.ErrorIs(t, rules.FileError(file), videos.ErrInvalidVideo, file)
	}
}

func TestAVideoIsKeptWithWhatTheBucketHoldsAndItsNameTrimmed(t *testing.T) {
	video, err := videos.Keep(mp4, "  Season 0 timelapse ", " 23 days. 1 planet. ", at, rules)
	require.NoError(t, err)

	assert.Equal(t, videos.VideoSnapshot{File: mp4, Name: "Season 0 timelapse", Caption: "23 days. 1 planet.", KeptAt: at},
		video.TakeSnapshot())
	assert.Equal(t, video, video.TakeSnapshot().Restore())
}

func TestAVideoNeedsANameAndACaptionThatFitsAPost(t *testing.T) {
	for name, caption := range map[string]string{
		"   ":                    "a caption",
		strings.Repeat("a", 101): "a caption",
		"a name":                 strings.Repeat("é", 2201),
	} {
		_, err := videos.Keep(mp4, name, caption, at, rules)
		require.ErrorIs(t, err, videos.ErrInvalidVideo)
	}

	_, err := videos.Keep(mp4, strings.Repeat("é", 100), strings.Repeat("é", 2200), at, rules)
	require.NoError(t, err, "counted in characters, not bytes")
}

func TestAFileTheBucketHoldsIsCheckedAgain(t *testing.T) {
	_, err := videos.Keep(videos.File{ID: mp4.ID, MediaType: "text/html", Size: 10}, "a name", "", at, rules)
	require.ErrorIs(t, err, videos.ErrInvalidVideo)
}

func TestADownloadIsSavedUnderTheNameInPlainLettersAndTheTypesExtension(t *testing.T) {
	assert.Equal(t, "Season-0-flat.mp4", videos.FileName("Season 0 flat", "video/mp4"))
	assert.Equal(t, "Final_Battle-3.mov", videos.FileName(" Final_Battle ×3!", "video/quicktime"))
	assert.Equal(t, "Algrie-TO-0.webm", videos.FileName("Algérie: TO 0%", "video/webm"))
	assert.Equal(t, "video.mp4", videos.FileName("🌍", "video/mp4"))
}

func TestAVideoIDIsAUUID(t *testing.T) {
	id, err := videos.VideoIDOf("6f1c2c64-6a4a-4b0e-9f43-2f0d7c1b5a01")
	require.NoError(t, err)
	assert.Equal(t, "6f1c2c64-6a4a-4b0e-9f43-2f0d7c1b5a01", id.String())

	for _, value := range []string{"", "a video", "00000000-0000-0000-0000-000000000000"} {
		_, err := videos.VideoIDOf(value)
		require.ErrorIs(t, err, videos.ErrInvalidVideoID, value)
	}
}
