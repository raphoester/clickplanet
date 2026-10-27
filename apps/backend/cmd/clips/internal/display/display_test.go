package display_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/clips/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

var (
	at         = time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC)
	everywhere = []clips.PlatformOutcome{{Platform: "tiktok"}, {Platform: "youtube"}}
)

type brokenPage struct{}

func (brokenPage) Write([]byte) (int, error) {
	return 0, errors.New("the disk is full")
}

func TestThePicksAreAJSONArrayOnOneLineOfEachClipsNumberStoryAndArms(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, display.Picks(&out, []clips.Assignment{
		{Pick: 1, Story: clips.Story{Seasonal: 1}, Variants: clips.Variants{"length": "short"}},
		{Pick: 2, Story: clips.Story{Front: 3, Kind: "battle", Attacker: "fr", Rival: "tr", Countries: []string{"tr"}, Headline: "FRANCE VS TURKEY"}, Variants: clips.Variants{"length": "as-chosen"}},
	}))

	assert.JSONEq(t, `[{"pick":1,"story":{"seasonal":1},"variants":{"length":"short"}},`+
		`{"pick":2,"story":{"front":3,"kind":"battle","attacker":"fr","rival":"tr","countries":["tr"],"headline":"FRANCE VS TURKEY"},"variants":{"length":"as-chosen"}}]`, out.String())
	assert.Equal(t, 1, strings.Count(out.String(), "\n"))
	assert.True(t, strings.HasSuffix(out.String(), "\n"))
}

func TestAPickInNoExperimentHasNoArmsNeverNull(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, display.Picks(&out, []clips.Assignment{{Pick: 3, Story: clips.Story{Tour: 1}}}))

	assert.JSONEq(t, `[{"pick":3,"story":{"tour":1},"variants":{}}]`, out.String())
}

func TestNoPickIsAnEmptyArrayNeverNull(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, display.Picks(&out, []clips.Assignment{}))

	assert.Equal(t, "[]\n", out.String())
}

func TestPicksThatCannotBeWrittenAreAnError(t *testing.T) {
	require.Error(t, display.Picks(brokenPage{}, []clips.Assignment{{Pick: 1}}))
}

func TestTheReportIsATableOfEachClipOnEachPlatformItsTimeAndWhatBecameOfIt(t *testing.T) {
	var out bytes.Buffer

	scheduled, err := display.Report("", &out, clips.Report{
		{Title: "KOSOVO IS ATTACKING THE MIDDLE EAST", At: at, Post: "post-1", Platforms: everywhere},
		{
			Title: "FRANCE IS ATTACKING EUROPE", Variants: clips.Variants{"length": "short"}, At: at.Add(2 * time.Hour), Post: "post-2",
			Platforms: []clips.PlatformOutcome{
				{Platform: "tiktok"}, {Platform: "youtube", Failure: "zernio answered 400 to POST /posts: a | b\nc"},
			},
		},
	})

	require.NoError(t, err)
	assert.False(t, scheduled)
	assert.Equal(t, `| Clip | Arms | Platform | Goes out | |
|---|---|---|---|---|
| KOSOVO IS ATTACKING THE MIDDLE EAST | none | tiktok | Mon 12 Oct, 16:40 UTC | scheduled |
| KOSOVO IS ATTACKING THE MIDDLE EAST | none | youtube | Mon 12 Oct, 16:40 UTC | scheduled |
| FRANCE IS ATTACKING EUROPE | length short | tiktok | Mon 12 Oct, 18:40 UTC | scheduled |
| FRANCE IS ATTACKING EUROPE | length short | youtube | Mon 12 Oct, 18:40 UTC | **not scheduled**: zernio answered 400 to POST /posts: a \| b c |
`, out.String())
}

// On the TikTok row alone: YouTube is sent the video as it is.
func TestTheReportSaysWhichTrendingTrackAClipGoesOutWithOnTikTokOrWhyItWentOutWithItsAnthem(t *testing.T) {
	var out bytes.Buffer
	sound := &clips.Sound{
		Track: clips.Track{ID: "clip-1", Excerpt: true, Title: "Song | One", Artist: "Singer", Rank: 4}, Country: "US", TrackVolume: 100,
	}

	scheduled, err := display.Report("", &out, clips.Report{
		{Title: "KOSOVO", Variants: clips.Variants{"sound": "trending"}, At: at, Post: "post-1", Platforms: everywhere, Sound: sound},
		{
			Title: "FRANCE", Variants: clips.Variants{"sound": "fell-back"}, At: at, Post: "post-2", Platforms: everywhere,
			SoundMissed: "the trending chart could not be read: zernio answered 400",
		},
	})

	require.NoError(t, err)
	assert.True(t, scheduled)
	assert.Equal(t, `| Clip | Arms | Platform | Goes out | |
|---|---|---|---|---|
| KOSOVO | sound trending | tiktok | Mon 12 Oct, 16:40 UTC | scheduled, with “Song \| One” by Singer (#4 of US's trending chart) |
| KOSOVO | sound trending | youtube | Mon 12 Oct, 16:40 UTC | scheduled |
| FRANCE | sound fell-back | tiktok | Mon 12 Oct, 16:40 UTC | scheduled, **with its anthem** instead of a trending track: the trending chart could not be read: zernio answered 400 |
| FRANCE | sound fell-back | youtube | Mon 12 Oct, 16:40 UTC | scheduled |
`, out.String())
}

func TestAReportWithEveryClipScheduledOnEveryPlatformSaysSo(t *testing.T) {
	var out bytes.Buffer

	scheduled, err := display.Report("", &out, clips.Report{{Title: "T", At: at, Post: "post-1", Platforms: everywhere}})

	require.NoError(t, err)
	assert.True(t, scheduled)
}

func TestAClipHeldForAFullDayIsARowOfItsOwnAndStillScheduled(t *testing.T) {
	var out bytes.Buffer

	scheduled, err := display.Report("", &out, clips.Report{
		{Title: "KOSOVO IS ATTACKING THE MIDDLE EAST", At: at, Post: "post-1", Platforms: everywhere},
		{Title: "FRANCE IS ATTACKING EUROPE", Variants: clips.Variants{"length": "short"}, Held: true},
	})

	require.NoError(t, err)
	assert.True(t, scheduled)
	assert.Contains(t, out.String(), "| FRANCE IS ATTACKING EUROPE | length short | none | | held: every platform has posted its day |\n")
}

func TestATimeIsSaidInUTCWhateverItsZone(t *testing.T) {
	var out bytes.Buffer
	sofia := time.FixedZone("EEST", 3*60*60)

	_, err := display.Report("", &out, clips.Report{{Title: "T", At: at.In(sofia), Post: "post-1", Platforms: everywhere}})

	require.NoError(t, err)
	assert.Contains(t, out.String(), "| Mon 12 Oct, 16:40 UTC |")
}

func TestAReportThatCannotBeWrittenIsAnErrorAndNotScheduled(t *testing.T) {
	scheduled, err := display.Report("", brokenPage{}, clips.Report{{Title: "T", At: at, Post: "post-1", Platforms: everywhere}})

	require.Error(t, err)
	assert.False(t, scheduled)
}

func TestTheReportIsAddedToTheSummaryFileWhenOneIsNamedAndNothingIsPrinted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "summary.md")
	require.NoError(t, os.WriteFile(path, []byte("what the workflow wrote before\n"), 0o600))
	var out bytes.Buffer

	scheduled, err := display.Report(path, &out, clips.Report{{Title: "T", At: at, Post: "post-1", Platforms: everywhere}})

	require.NoError(t, err)
	assert.True(t, scheduled)
	written, err := os.ReadFile(path) //nolint:gosec // the file the test wrote.
	require.NoError(t, err)
	assert.Contains(t, string(written), "what the workflow wrote before\n| Clip |")
	assert.Empty(t, out.String())
}

func TestASummaryFileThatCannotBeOpenedIsAnErrorAndNotScheduled(t *testing.T) {
	var out bytes.Buffer

	scheduled, err := display.Report(filepath.Join(t.TempDir(), "no-such-folder", "summary.md"), &out, clips.Report{{Title: "T", At: at, Post: "post-1", Platforms: everywhere}})

	require.Error(t, err)
	assert.False(t, scheduled)
	assert.Empty(t, out.String())
}
