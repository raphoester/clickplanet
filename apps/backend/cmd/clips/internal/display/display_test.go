package display_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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

func TestThePicksAreAJSONArrayOnOneLine(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, display.Picks(&out, []clips.Pick{1, 2}))

	assert.Equal(t, "[1,2]\n", out.String())
}

func TestNoPickIsAnEmptyArrayNeverNull(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, display.Picks(&out, []clips.Pick{}))

	assert.Equal(t, "[]\n", out.String())
}

func TestPicksThatCannotBeWrittenAreAnError(t *testing.T) {
	require.Error(t, display.Picks(brokenPage{}, []clips.Pick{1}))
}

func TestTheReportIsATableOfEachClipOnEachPlatformItsTimeAndWhatBecameOfIt(t *testing.T) {
	var out bytes.Buffer

	scheduled, err := display.Report("", &out, clips.Report{
		{Title: "KOSOVO IS ATTACKING THE MIDDLE EAST", At: at, Post: "post-1", Platforms: everywhere},
		{Title: "FRANCE IS ATTACKING EUROPE", At: at.Add(2 * time.Hour), Post: "post-2", Platforms: []clips.PlatformOutcome{
			{Platform: "tiktok"}, {Platform: "youtube", Failure: "zernio answered 400 to POST /posts: a | b\nc"},
		}},
	})

	require.NoError(t, err)
	assert.False(t, scheduled)
	assert.Equal(t, `| Clip | Platform | Goes out | |
|---|---|---|---|
| KOSOVO IS ATTACKING THE MIDDLE EAST | tiktok | Mon 12 Oct, 16:40 UTC | scheduled |
| KOSOVO IS ATTACKING THE MIDDLE EAST | youtube | Mon 12 Oct, 16:40 UTC | scheduled |
| FRANCE IS ATTACKING EUROPE | tiktok | Mon 12 Oct, 18:40 UTC | scheduled |
| FRANCE IS ATTACKING EUROPE | youtube | Mon 12 Oct, 18:40 UTC | **not scheduled**: zernio answered 400 to POST /posts: a \| b c |
`, out.String())
}

func TestAReportWithEveryClipScheduledOnEveryPlatformSaysSo(t *testing.T) {
	var out bytes.Buffer

	scheduled, err := display.Report("", &out, clips.Report{{Title: "T", At: at, Post: "post-1", Platforms: everywhere}})

	require.NoError(t, err)
	assert.True(t, scheduled)
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
