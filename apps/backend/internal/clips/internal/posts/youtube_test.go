package posts_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func TestAShortIsTitledWithTheTeaserAndSaysHowToPlayAndItsTags(t *testing.T) {
	words := captionOf(t, portugalInItaly(t, ""), 0).YouTube

	assert.Equal(t, posts.YouTubeWords{
		Title: "ITALY RETAKES ITS LAND FROM PORTUGAL | CLICKPLANET",
		Description: "Join the battle now 👉 clickplanet.lol\n" +
			"#clickplanet #rplace #wplace #pixelwar #map #geographygame #conquestgame #videogame #clickergame #viral " +
			"#europe #italy #portugal",
		Tags: []string{
			"clickplanet", "rplace", "wplace", "pixelwar", "map", "geographygame", "conquestgame", "videogame", "clickergame", "viral",
			"europe", "italy", "portugal",
		},
	}, words)
}

func TestAShortKeepsTheCreditOfTheMusic(t *testing.T) {
	words := captionOf(t, portugalInItaly(t, "Fida'i (instrumental) by Sounds Monarch, CC BY 3.0"), 0).YouTube

	assert.Contains(t, words.Description, "clickplanet.lol\n🎵 Fida'i (instrumental) by Sounds Monarch, CC BY 3.0\n#clickplanet ")
}

func TestAShortHasNoTikTokTagAndNoShortsTag(t *testing.T) {
	words := captionOf(t, portugalInItaly(t, ""), 0).YouTube

	assert.NotContains(t, words.Description, "tiktok")
	assert.NotContains(t, strings.ToLower(words.Description), "#shorts")
}

func TestATitleTooLongForYouTubeIsTheTeaserCut(t *testing.T) {
	teaser := strings.Repeat("SAINT VINCENT AND THE GRENADINES ", 4)
	restored, err := posts.ClipSnapshot{
		Pick: 1, Title: "T", Key: posts.StoryKey{Flag: "vc", Continent: "the world"}, Teasers: []string{teaser}, Video: "clip-1.mp4",
	}.Restore()
	require.NoError(t, err)

	title := captionOf(t, restored, 0).YouTube.Title

	assert.Equal(t, 100, utf8.RuneCountInString(title))
	assert.Equal(t, string([]rune(teaser)[:99])+"…", title)
}

func TestATeaserThatFitsOnlyWithoutTheBrandLosesTheBrand(t *testing.T) {
	teaser := strings.Repeat("A", 90)
	restored, err := posts.ClipSnapshot{
		Pick: 1, Title: "T", Key: posts.StoryKey{Flag: "fr", Continent: "Europe"}, Teasers: []string{teaser}, Video: "clip-1.mp4",
	}.Restore()
	require.NoError(t, err)

	assert.Equal(t, teaser, captionOf(t, restored, 0).YouTube.Title)
}

func manyCountries(t *testing.T, count int, tag string) posts.Clip {
	t.Helper()

	countries := make([]posts.TaggedCountry, 0, count)
	for i := range count {
		countries = append(countries, posts.TaggedCountry{Code: "c" + string(rune('a'+i%26)), Tag: tag + strings.Repeat("x", i)})
	}
	restored, err := posts.ClipSnapshot{
		Pick: 1, Title: "T", Key: posts.StoryKey{Flag: "fr", Continent: "the world"}, Teasers: []string{"T"},
		Countries: countries, Video: "clip-1.mp4",
	}.Restore()
	require.NoError(t, err)
	return restored
}

func TestAShortHasNoMoreHashtagsThanYouTubeReads(t *testing.T) {
	words := captionOf(t, manyCountries(t, 80, "c"), 0).YouTube

	lines := strings.Split(words.Description, "\n")
	assert.Equal(t, 60, strings.Count(lines[len(lines)-1], "#"))
}

func TestTheTagsOfAShortStopBeforeYouTubesLength(t *testing.T) {
	words := captionOf(t, manyCountries(t, 40, "country"), 0).YouTube

	assert.LessOrEqual(t, utf8.RuneCountInString(strings.Join(words.Tags, ",")), 500)
	assert.Less(t, len(words.Tags), 50, "some countries are left out")
	assert.Equal(t, "clickplanet", words.Tags[0])
}
