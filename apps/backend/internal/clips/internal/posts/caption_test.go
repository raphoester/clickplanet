package posts_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func portugalInItaly(t *testing.T, attribution string) posts.Clip {
	t.Helper()

	restored, err := posts.ClipSnapshot{
		Pick: 1, Title: "PORTUGAL GETS KICKED OUT OF ITALY", Key: posts.StoryKey{Flag: "pt", Continent: "Europe"},
		Teasers:      []string{"ITALY RETAKES ITS LAND FROM PORTUGAL", "GENERATIONAL DOWNFALL OF PORTUGAL"},
		Countries:    []posts.TaggedCountry{{Code: "it", Tag: "italy"}, {Code: "pt", Tag: "portugal"}},
		ContinentTag: "europe", Attribution: attribution, Video: "clip-1.mp4",
	}.Restore()
	require.NoError(t, err)
	return restored
}

func captionOf(t *testing.T, clip posts.Clip, face int) posts.PostSnapshot {
	t.Helper()

	return plan.Posts([]posts.Clip{clip}, time.Date(2026, 10, 14, 12, 17, 0, 0, time.UTC), posts.LoadedDice(face))[0].TakeSnapshot()
}

func TestACaptionSaysWhereItHappensHowToPlayAndItsTags(t *testing.T) {
	post := captionOf(t, portugalInItaly(t, ""), 0)

	assert.Equal(t, "ITALY RETAKES ITS LAND FROM PORTUGAL | CLICKPLANET\n"+
		"Join the battle now 👉 https://clickplanet.lol\n"+
		"#clickplanet #rplace #wplace #pixelwar #map #geographygame #conquestgame #videogame #clickergame #viral "+
		"#europe #italy #portugal #portugaltiktok🇵🇹", post.TikTok.Caption)
	assert.Equal(t, "join-the-battle", post.CallToAction)
}

func TestTheTeaserAndTheCallAreTheDicesPick(t *testing.T) {
	post := captionOf(t, portugalInItaly(t, ""), 3)

	assert.Contains(t, post.TikTok.Caption, "GENERATIONAL DOWNFALL OF PORTUGAL | CLICKPLANET\nPlay for free 👉 https://clickplanet.lol\n")
	assert.Equal(t, "free", post.CallToAction)
}

func TestTheCreditOfTheMusicComesBeforeTheTags(t *testing.T) {
	post := captionOf(t, portugalInItaly(t, "Fida'i (instrumental) by Sounds Monarch, CC BY 3.0"), 0)

	assert.Contains(t, post.TikTok.Caption, "https://clickplanet.lol\n🎵 Fida'i (instrumental) by Sounds Monarch, CC BY 3.0\n#clickplanet ")
}

func TestOnlyTheFlagTheClipIsAboutGetsItsTikTokTag(t *testing.T) {
	restored, err := posts.ClipSnapshot{
		Pick: 1, Title: "KOSOVO IS ATTACKING THE MIDDLE EAST", Key: posts.StoryKey{Flag: "xk", Continent: "Middle East"},
		Teasers:   []string{"KOSOVO TAKES OVER THE MIDDLE EAST"},
		Countries: []posts.TaggedCountry{{Code: "xk", Tag: "kosovo"}, {Code: "us", Tag: "usa"}}, Video: "clip-1.mp4",
	}.Restore()
	require.NoError(t, err)

	caption := captionOf(t, restored, 0).TikTok.Caption

	assert.Contains(t, caption, "#viral #kosovo #usa #kosovotiktok")
	assert.NotContains(t, caption, "#usatiktok")
	assert.True(t, strings.HasSuffix(caption, "#kosovotiktok"), "a private-use code has no flag")
}

func TestAnAmericanStoryGetsItsFlag(t *testing.T) {
	restored, err := posts.ClipSnapshot{
		Pick: 1, Title: "THE USA IS INVADING CANADA", Key: posts.StoryKey{Flag: "us", Continent: "North America"},
		Teasers: []string{"THE USA INVADES CANADA"}, Countries: []posts.TaggedCountry{{Code: "us", Tag: "usa"}}, Video: "clip-1.mp4",
	}.Restore()
	require.NoError(t, err)

	assert.Contains(t, captionOf(t, restored, 0).TikTok.Caption, "#usa #usatiktok🇺🇸")
}

func franceVsTurkey(t *testing.T) posts.Clip {
	t.Helper()

	restored, err := posts.ClipSnapshot{
		Pick: 1, Title: "FRANCE VS TURKEY", Key: posts.StoryKey{Flag: "fr", Continent: "Middle East"},
		Teasers: []string{"WAR FOR TURKEY"}, Countries: []posts.TaggedCountry{{Code: "fr", Tag: "france"}, {Code: "tr", Tag: "turkey"}},
		Video: "clip-1.mp4", Cast: "5f0c2a11-0b6d-4f7e-9a51-3f1f8f0c2a11",
		Fighters: []posts.ClipFighter{{Number: 1, Name: "BraveFox42", Flag: "fr"}, {Number: 2, Name: "Ankara 55", Flag: "tr"}, {Number: 3, Name: "Kosovar", Flag: "xk"}},
	}.Restore()
	require.NoError(t, err)
	return restored
}

func TestTheFightersAClipNamesFollowItsTeaser(t *testing.T) {
	post := captionOf(t, franceVsTurkey(t), 0)

	assert.True(t, strings.HasPrefix(post.TikTok.Caption, "WAR FOR TURKEY | CLICKPLANET\n"+
		"⚔️ Top fighters: BraveFox42 🇫🇷, Ankara 55 🇹🇷, Kosovar\n"+
		"Join the battle now 👉 https://clickplanet.lol\n"), post.TikTok.Caption)
	assert.True(t, strings.HasPrefix(post.YouTube.Description, "⚔️ Top fighters: BraveFox42 🇫🇷, Ankara 55 🇹🇷, Kosovar\n"+
		"Join the battle now"), post.YouTube.Description)
}

func TestAClipThatNamesNobodyHasNoLineOfFighters(t *testing.T) {
	assert.NotContains(t, captionOf(t, portugalInItaly(t, ""), 0).TikTok.Caption, "Top fighters")
}
