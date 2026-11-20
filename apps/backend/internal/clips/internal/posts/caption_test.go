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
		"Join the battle now 👉 https://clickplanet.lol/t\n"+
		"#clickplanet #rplace #wplace #pixelwar #map #geographygame #conquestgame #videogame #clickergame #viral "+
		"#europe #italy #portugal #portugaltiktok🇵🇹", post.TikTok.Caption)
	assert.Equal(t, "join-the-battle", post.CallToAction)
}

func TestTheTeaserAndTheCallAreTheDicesPick(t *testing.T) {
	post := captionOf(t, portugalInItaly(t, ""), 3)

	assert.Contains(t, post.TikTok.Caption, "GENERATIONAL DOWNFALL OF PORTUGAL | CLICKPLANET\nPlay for free 👉 https://clickplanet.lol/t\n")
	assert.Equal(t, "free", post.CallToAction)
}

func TestTheCreditOfTheMusicComesBeforeTheTags(t *testing.T) {
	post := captionOf(t, portugalInItaly(t, "Fida'i (instrumental) by Sounds Monarch, CC BY 3.0"), 0)

	assert.Contains(t, post.TikTok.Caption, "https://clickplanet.lol/t\n🎵 Fida'i (instrumental) by Sounds Monarch, CC BY 3.0\n#clickplanet ")
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
		"Join the battle now 👉 https://clickplanet.lol/t\n"), post.TikTok.Caption)
	assert.True(t, strings.HasPrefix(post.YouTube.Description, "⚔️ Top fighters: BraveFox42 🇫🇷, Ankara 55 🇹🇷, Kosovar\n"+
		"Join the battle now"), post.YouTube.Description)
}

func TestAClipThatNamesNobodyHasNoLineOfFighters(t *testing.T) {
	assert.NotContains(t, captionOf(t, portugalInItaly(t, ""), 0).TikTok.Caption, "Top fighters")
}

func ownLandCarousel(t *testing.T) posts.Clip {
	t.Helper()

	restored, err := posts.ClipSnapshot{
		Pick: 1, Title: "THESE 5 COUNTRIES DON'T OWN THEIR OWN LAND",
		Key:         posts.StoryKey{Flag: "fr", Continent: posts.TheWorld, Angle: posts.Listed},
		Teasers:     []string{"THESE 5 COUNTRIES DON'T OWN THEIR OWN LAND"},
		TeaserNames: []string{"editor"},
		Countries: []posts.TaggedCountry{
			{Code: "fr", Tag: "france"},
			{Code: "tr", Tag: "turkey"},
			{Code: "in", Tag: "india"},
			{Code: "de", Tag: "germany"},
			{Code: "br", Tag: "brazil"},
		},
		Video: "clip-1.mp4", Experiment: "carousel", List: posts.OwnLand,
		Photos: []posts.PhotoFile{"clip-1-1.jpg", "clip-1-2.jpg", "clip-1-3.jpg", "clip-1-4.jpg", "clip-1-5.jpg", "clip-1-6.jpg", "clip-1-7.jpg"},
	}.Restore()
	require.NoError(t, err)
	return restored
}

func TestACarouselsTitleIsItsCoverAndItsCaptionAsksWhereTheViewersCountryIsWithNoCallToPlay(t *testing.T) {
	post := captionOf(t, ownLandCarousel(t), 1)

	assert.Equal(t, posts.TikTokWords{
		Title: "THESE 5 COUNTRIES DON'T OWN THEIR OWN LAND",
		Caption: "Where is your country? 👇\n" +
			"https://clickplanet.lol/t\n" +
			"#clickplanet #rplace #wplace #pixelwar #map #geographygame #conquestgame #videogame #clickergame #viral " +
			"#france #turkey #india #germany #brazil #francetiktok🇫🇷",
	}, post.TikTok)
	assert.Empty(t, post.CallToAction)
	assert.Equal(t, "editor", post.Teaser)
	assert.Equal(t, posts.OwnLand, post.List)
	assert.Len(t, post.Photos, 7)
	assert.Equal(t, posts.ExperimentName("carousel"), post.Experiment)
}

func TestAClipsTikTokPostHasNoTitleOverItsVideo(t *testing.T) {
	assert.Empty(t, captionOf(t, portugalInItaly(t, ""), 0).TikTok.Title)
}

func TestACarouselIsItsListsPhotosUnderATitleTikTokTakes(t *testing.T) {
	carousel := ownLandCarousel(t).TakeSnapshot()
	changed := func(change func(*posts.ClipSnapshot)) posts.ClipSnapshot {
		snapshot := carousel
		change(&snapshot)
		return snapshot
	}

	for why, snapshot := range map[string]posts.ClipSnapshot{
		"no list":            changed(func(s *posts.ClipSnapshot) { s.List = "" }),
		"a list of no name":  changed(func(s *posts.ClipSnapshot) { s.List = "rankings" }),
		"no photo":           changed(func(s *posts.ClipSnapshot) { s.Photos = []posts.PhotoFile{} }),
		"a photo of no file": changed(func(s *posts.ClipSnapshot) { s.Photos = []posts.PhotoFile{"clip-1-1.jpg", ""} }),
		"36 photos": changed(func(s *posts.ClipSnapshot) {
			s.Photos = make([]posts.PhotoFile, 36)
			for i := range s.Photos {
				s.Photos[i] = "clip-1-1.jpg"
			}
		}),
		"a title over 90 letters": changed(func(s *posts.ClipSnapshot) { s.Title = strings.Repeat("A", 91) }),
	} {
		_, err := snapshot.Restore()
		assert.ErrorIs(t, err, posts.ErrInvalidClip, why)
	}
}
