package metadata_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/metadata"
)

var at = time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC)

// What a post of a clip with no record and no teaser names keeps.
var unnamedPost = map[string]any{
	"teaser": "unnamed", "callToAction": "join-the-battle", "scheduledAt": "2026-10-12T16:40:00Z",
	"runPosition": float64(1), "runSize": float64(1),
}

// variants: the arms of the clip, which the generator also wrote into its record.
func snapshotOf(t *testing.T, key posts.StoryKey, record posts.ClipRecord, teaser string, variants posts.Variants) posts.PostSnapshot {
	t.Helper()

	var names []string
	if teaser != "" {
		names = []string{teaser}
	}
	clip, err := posts.ClipSnapshot{
		Pick: 1, Title: "FRANCE IS ATTACKING EUROPE", Teasers: []string{"FRANCE TAKES OVER EUROPE"},
		Countries: []posts.TaggedCountry{{Code: "fr", Tag: "france"}}, ContinentTag: "europe", Key: key,
		Video: "clip-1.mp4", Record: record, TeaserNames: names, Variants: variants,
	}.Restore()
	require.NoError(t, err)

	post := posts.Plan{Platforms: []posts.Platform{posts.TikTok, posts.YouTube}}.Post(clip, 1, 1, posts.LoadedDice(0))
	return post.At(posts.Slot{At: at, Platforms: []posts.Platform{posts.TikTok, posts.YouTube}}).TakeSnapshot()
}

func TestWhatAStoryTellsOfItsFlagAndItsDayAreKeptWithThePost(t *testing.T) {
	kept, err := metadata.Of(snapshotOf(t, posts.StoryKey{Flag: "fr", Continent: "the world", Angle: posts.DayWon, Round: "1.5"}, nil, "", nil))

	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"storyFlag": "fr", "storyContinent": "the world", "storyAngle": "round", "storyRound": "1.5",
		"storyCountries": []string{"fr"}, "callToAction": "join-the-battle",
		"record": map[string]any{"schema": "clips.v1.PostRecord", "post": unnamedPost},
	}, kept)
}

func TestWhatThePostWasIsOneDocumentUnderOneKeyWithTheClipAsTheGeneratorWroteItAndTheTimePlanned(t *testing.T) {
	record := posts.ClipRecord(`{"format":"globe-replay","generator":"8dc31d2","common":{"durationSeconds":17.4,"music":"none",` +
		`"language":"en"},"globe":{"camera":"tour","look":"none","tilesChanged":5200,"tourStops":3,"storyEndedAt":"2026-10-12T14:12:30Z"},` +
		`"variants":{"end-card":"podium"}}`)

	kept, err := metadata.Of(snapshotOf(t, posts.StoryKey{Flag: "pt", Continent: "the world"}, record, "takes-over-the-world", nil))

	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"schema": "clips.v1.PostRecord",
		"clip": map[string]any{
			"format": "globe-replay", "generator": "8dc31d2",
			"common": map[string]any{"durationSeconds": 17.4, "music": "none", "language": "en"},
			"globe": map[string]any{
				"camera": "tour", "look": "none", "tilesChanged": float64(5200), "tourStops": float64(3), "storyEndedAt": "2026-10-12T14:12:30Z",
			},
			"variants": map[string]any{"end-card": "podium"},
		},
		"post": map[string]any{
			"teaser": "takes-over-the-world", "callToAction": "join-the-battle", "scheduledAt": "2026-10-12T16:40:00Z",
			"runPosition": float64(1), "runSize": float64(1),
		},
	}, kept["record"])
	assert.Equal(t, "pt", kept["storyFlag"])
	assert.NotContains(t, kept, "storyAngle", "a post of what a flag did")
}

func TestARecordThatIsNotAClipRecordIsAnError(t *testing.T) {
	_, err := metadata.Of(snapshotOf(t, posts.StoryKey{Flag: "fr", Continent: "Europe"}, posts.ClipRecord(`{"format":3}`), "invades", nil))

	require.ErrorContains(t, err, "not a clips.v1.ClipRecord")
}

// The length acts when the clip is rendered, not when it is posted: a clip in the control, or in the short arm, keeps
// what it kept before the experiments, its record aside, which carries its arms as the generator wrote them.
func TestAClipInAnArmOfTheLengthExperimentKeepsWhatAClipInNoneKeeps(t *testing.T) {
	record := posts.ClipRecord(`{"format":"globe-replay","variants":{"length":"as-chosen"}}`)
	kept := func(variants posts.Variants) map[string]any {
		of, err := metadata.Of(snapshotOf(t, posts.StoryKey{Flag: "fr", Continent: "Europe"}, record, "takes-over", variants))
		require.NoError(t, err)
		return of
	}

	before := kept(nil)

	assert.Equal(t, before, kept(posts.Variants{"length": "as-chosen"}))
	assert.Equal(t, before, kept(posts.Variants{"length": "short"}))
}

func TestAKeyIsReadBackWithItsAngleAndAPostWithNoneHasNone(t *testing.T) {
	key, keyed := metadata.KeyOf(map[string]any{"storyFlag": "fr", "storyContinent": "the world", "storyAngle": "round", "storyRound": "1.5"})
	assert.True(t, keyed)
	assert.Equal(t, posts.StoryKey{Flag: "fr", Continent: "the world", Angle: posts.DayWon, Round: "1.5"}, key)

	key, keyed = metadata.KeyOf(map[string]any{"storyFlag": "hr", "storyContinent": "Europe", "storyAngle": "holding"})
	assert.True(t, keyed)
	assert.Equal(t, posts.StoryKey{Flag: "hr", Continent: "Europe", Angle: posts.Holding}, key)

	for _, none := range []map[string]any{nil, {}, {"storyFlag": "fr"}, {"storyContinent": "Europe"}, {"storyFlag": 3, "storyContinent": "Europe"}} {
		_, keyed := metadata.KeyOf(none)
		assert.False(t, keyed, "%v", none)
	}
}

func TestTheCountriesReadBackAreTheCodesAlone(t *testing.T) {
	assert.Equal(t, []string{"kz", "ps", "ro"}, metadata.CountriesOf(map[string]any{"storyCountries": []any{"kz", "ps", float64(3), "", "ro"}}))
	assert.Empty(t, metadata.CountriesOf(map[string]any{"storyCountries": "pt"}))
	assert.Empty(t, metadata.CountriesOf(nil))
}
