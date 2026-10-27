package generator_output_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/generator_output"
)

const (
	writtenFindings = `{
  "since": "2026-10-26T08:29:17Z",
  "until": "2026-10-26T20:29:17Z",
  "fronts": [
    {
      "scope": "tr",
      "since": "2026-10-26T15:03:00Z",
      "until": "2026-10-26T20:29:17Z",
      "tilesChanged": 9844,
      "captured": 9000,
      "countries": [
        {"code": "tr", "continent": "Middle East", "tiles": 2500, "changed": 6000, "holdersStart": {"tr": 2000}, "holdersEnd": {"fr": 1500, "tr": 900}, "takers": {"fr": 5000}}
      ],
      "flags": [
        {"code": "fr", "continent": "Europe", "took": 5000, "filled": 100, "lost": 3000, "tookFrom": {"tr": 4800}, "solidity": 0.86}
      ],
      "bombs": 3,
      "score": 3886.5
    }
  ],
  "tours": [
    {"flag": "fr", "since": "2026-10-26T08:29:17Z", "until": "2026-10-26T20:29:17Z", "took": 27992, "byContinent": {"Europe": 12000}, "tookFrom": {"dz": 8000}, "nextTook": 9000, "solidity": 0.95, "bombs": 12, "flippedByContinent": {"South Asia": 6246}}
  ],
  "places": [
    {"kind": "continent", "continent": "South Asia", "tiles": 9600, "holder": "fr", "held": 6170, "share": 0.643, "rivals": {"dz": 1200}, "changed": 6000}
  ],
  "seasonal": [
    {"title": "PALAU WINS DAY 18", "key": {"flag": "pw", "continent": "the world", "angle": "round", "round": "0.18"}, "countries": ["pw", "dz"], "teasers": ["PALAU TAKES DAY 18"], "facts": {"kind": "round", "storyTilesChanged": 1200}}
  ],
  "names": {"fr": "France", "tr": "Turkey"},
  "continents": [{"name": "Europe", "flag": "eu"}, {"name": "Middle East"}]
}
`
	writtenClip = `{
  "pick": 2,
  "title": "FRANCE IS ATTACKING EUROPE",
  "key": {
    "flag": "fr",
    "continent": "Europe"
  },
  "teasers": [
    "FRANCE TAKES OVER EUROPE",
    "EUROPE IS UNDER ATTACK FROM FRANCE"
  ],
  "countries": [
    {
      "code": "fr",
      "tag": "france"
    },
    {
      "code": "de",
      "tag": "germany"
    }
  ],
  "continentTag": "europe",
  "credit": "Fida'i (instrumental) by Sounds Monarch, CC BY 3.0"
}
`
)

func write(t *testing.T, folder, name, content string) string {
	t.Helper()

	path := filepath.Join(folder, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestTheFindingsAreEveryFrontTourPlaceAndSeasonalStoryTheGeneratorWrote(t *testing.T) {
	file := write(t, t.TempDir(), "findings.json", writtenFindings)

	findings, err := generator_output.FindingsFile(file).Findings(t.Context())

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 26, 8, 29, 17, 0, time.UTC), findings.Since)
	assert.Equal(t, []posts.Front{{
		Scope: "tr", Since: time.Date(2026, 10, 26, 15, 3, 0, 0, time.UTC), Until: time.Date(2026, 10, 26, 20, 29, 17, 0, time.UTC),
		TilesChanged: 9844, Captured: 9000,
		Countries: []posts.FrontCountry{{
			Code: "tr", Continent: "Middle East", Tiles: 2500, Changed: 6000, HoldersStart: map[string]int{"tr": 2000},
			HoldersEnd: map[string]int{"fr": 1500, "tr": 900}, Takers: map[string]int{"fr": 5000},
		}},
		Flags: []posts.FrontFlag{{Code: "fr", Continent: "Europe", Took: 5000, Filled: 100, Lost: 3000, TookFrom: map[string]int{"tr": 4800}, Solidity: 0.86}},
		Bombs: 3, Score: 3886.5,
	}}, findings.Fronts)
	assert.Equal(t, []posts.Tour{{
		Flag: "fr", Since: findings.Since, Until: findings.Until, Took: 27992, ByContinent: map[string]int{"Europe": 12000},
		TookFrom: map[string]int{"dz": 8000}, NextTook: 9000, Solidity: 0.95, Bombs: 12, FlippedByContinent: map[string]int{"South Asia": 6246},
	}}, findings.Tours)
	assert.Equal(t, []posts.Place{{
		Kind: posts.ContinentPlace, Continent: "South Asia", Tiles: 9600, Holder: "fr", Held: 6170, Share: 0.643,
		Rivals: map[string]int{"dz": 1200}, Changed: 6000,
	}}, findings.Places)
	require.Len(t, findings.Seasonal, 1)
	assert.Equal(t, "PALAU WINS DAY 18", findings.Seasonal[0].Title)
	assert.Equal(t, posts.StoryKey{Flag: "pw", Continent: "the world", Angle: posts.DayWon, Round: "0.18"}, findings.Seasonal[0].Key)
	assert.Equal(t, []string{"PALAU TAKES DAY 18"}, findings.Seasonal[0].Teasers)
	assert.JSONEq(t, `{"kind":"round","storyTilesChanged":1200}`, string(findings.Seasonal[0].Facts))
	assert.Equal(t, map[string]string{"fr": "France", "tr": "Turkey"}, findings.Names)
	assert.Equal(t, map[string]string{"Europe": "eu", "Middle East": ""}, findings.Continents)
}

func TestFindingsWithNothingFoundAreEmpty(t *testing.T) {
	file := write(t, t.TempDir(), "findings.json", "{}\n")

	findings, err := generator_output.FindingsFile(file).Findings(t.Context())

	require.NoError(t, err)
	assert.False(t, findings.Any())
}

func TestFindingsThatAreMissingOrAreNotFindingsAreAnError(t *testing.T) {
	folder := t.TempDir()

	_, err := generator_output.FindingsFile(filepath.Join(folder, "gone.json")).Findings(t.Context())
	require.Error(t, err)

	_, err = generator_output.FindingsFile(write(t, folder, "findings.json", "saved 8 fronts")).Findings(t.Context())
	require.ErrorContains(t, err, "are not a clips.v1.Findings")
}

func TestAStoryIsWrittenAsTheGeneratorsStoryFlagTakesIt(t *testing.T) {
	front, err := generator_output.StoryJSON(posts.Story{
		Front: 3, Kind: posts.Battle, Attacker: "fr", Rival: "tr", TakenFrom: []string{"tr"}, Countries: []string{"tr", "sy"}, Headline: "FRANCE VS TURKEY",
	})
	require.NoError(t, err)
	seasonal, err := generator_output.StoryJSON(posts.Story{Seasonal: 1})
	require.NoError(t, err)

	assert.JSONEq(t, `{"front":3,"kind":"battle","attacker":"fr","rival":"tr","takenFrom":["tr"],"countries":["tr","sy"],"headline":"FRANCE VS TURKEY"}`, string(front))
	assert.JSONEq(t, `{"seasonal":1}`, string(seasonal))
}

func TestAFolderGivesEachClipWithItsVideo(t *testing.T) {
	folder := t.TempDir()
	write(t, folder, "clip-2.json", writtenClip)
	video := write(t, folder, "clip-2.mp4", "the video")
	write(t, folder, "clip-2.txt", "the caption, for a person")

	clips, err := generator_output.Folder(folder).Rendered(t.Context())

	require.NoError(t, err)
	require.Len(t, clips, 1)
	assert.Equal(t, posts.ClipSnapshot{
		Pick:    2,
		Title:   "FRANCE IS ATTACKING EUROPE",
		Key:     posts.StoryKey{Flag: "fr", Continent: "Europe"},
		Teasers: []string{"FRANCE TAKES OVER EUROPE", "EUROPE IS UNDER ATTACK FROM FRANCE"},
		Countries: []posts.TaggedCountry{
			{Code: "fr", Tag: "france"},
			{Code: "de", Tag: "germany"},
		},
		ContinentTag: "europe",
		Attribution:  "Fida'i (instrumental) by Sounds Monarch, CC BY 3.0",
		Video:        posts.VideoFile(video),
	}, clips[0].TakeSnapshot())
}

func TestAClipKeepsWhatItWasAndTheNamesOfItsTeasersAndOneWrittenBeforeThatHasNeither(t *testing.T) {
	folder := t.TempDir()
	write(t, folder, "clip-1.json", `{"pick": 1, "title": "FRANCE VS PALAU", "key": {"flag": "fr", "continent": "Europe"},
		"teasers": ["WAR FOR FRANCE: FRANCE VS PALAU", "FRANCE AND PALAU FIGHT FOR FRANCE"], "teaserNames": ["war-for", "fight-for"],
		"record": {"format": "globe-replay", "generator": "8dc31d2", "common": {"durationSeconds": 12.37, "music": "fr"},
		"globe": {"camera": "front", "tilesChanged": 1480}, "addedLater": true}}`)
	write(t, folder, "clip-1.mp4", "the video")
	write(t, folder, "clip-2.json", writtenClip)
	write(t, folder, "clip-2.mp4", "the video")

	clips, err := generator_output.Folder(folder).Rendered(t.Context())

	require.NoError(t, err)
	require.Len(t, clips, 2)
	named := clips[0].TakeSnapshot()
	assert.Equal(t, []string{"war-for", "fight-for"}, named.TeaserNames)
	assert.JSONEq(t, `{"format": "globe-replay", "generator": "8dc31d2", "common": {"durationSeconds": 12.37, "music": "fr"},
		"globe": {"camera": "front", "tilesChanged": 1480}}`, string(named.Record))
	assert.Empty(t, clips[1].TakeSnapshot().TeaserNames)
	assert.Empty(t, clips[1].TakeSnapshot().Record)
}

func TestAFolderWithNoClipGivesNone(t *testing.T) {
	clips, err := generator_output.Folder(t.TempDir()).Rendered(t.Context())

	require.NoError(t, err)
	assert.Empty(t, clips)
}

func TestAClipWithNoVideoBesideItIsAnError(t *testing.T) {
	folder := t.TempDir()
	write(t, folder, "clip-2.json", writtenClip)

	_, err := generator_output.Folder(folder).Rendered(t.Context())

	require.ErrorIs(t, err, generator_output.ErrNoVideo)
	assert.ErrorContains(t, err, "clip-2.mp4")
}

func TestAClipTheGeneratorLeftAPartOutOfIsAnError(t *testing.T) {
	folder := t.TempDir()
	write(t, folder, "clip-1.json", `{"pick": 1, "title": "FRANCE IS ATTACKING EUROPE", "key": {"flag": "fr", "continent": "Europe"}}`)
	write(t, folder, "clip-1.mp4", "the video")

	_, err := generator_output.Folder(folder).Rendered(t.Context())

	require.ErrorIs(t, err, posts.ErrInvalidClip)
	assert.ErrorContains(t, err, "clip-1.json")
}

func TestAFileThatIsNotAClipIsAnError(t *testing.T) {
	folder := t.TempDir()
	write(t, folder, "clip-1.json", "not json")

	_, err := generator_output.Folder(folder).Rendered(t.Context())

	require.ErrorContains(t, err, "clip-1.json is not a clips.v1.Clip")
}

func TestAClipKeepsTheArmsItWasMadeIn(t *testing.T) {
	folder := t.TempDir()
	write(t, folder, "clip-1.json", `{"pick": 1, "title": "FRANCE VS PALAU", "key": {"flag": "fr", "continent": "Europe"},
		"teasers": ["WAR FOR FRANCE"], "record": {"format": "globe-replay", "variants": {"length": "short"}}}`)
	write(t, folder, "clip-1.mp4", "the video")
	write(t, folder, "clip-2.json", writtenClip)
	write(t, folder, "clip-2.mp4", "the video")

	clips, err := generator_output.Folder(folder).Rendered(t.Context())

	require.NoError(t, err)
	require.Len(t, clips, 2)
	assert.Equal(t, posts.Variants{"length": "short"}, clips[0].TakeSnapshot().Variants)
	assert.JSONEq(t, `{"format": "globe-replay", "variants": {"length": "short"}}`, string(clips[0].TakeSnapshot().Record))
	assert.Empty(t, clips[1].TakeSnapshot().Variants)
}
