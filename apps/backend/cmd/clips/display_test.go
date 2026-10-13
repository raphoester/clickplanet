package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

var at = time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC)

type brokenPage struct{}

func (brokenPage) Write([]byte) (int, error) {
	return 0, errors.New("the disk is full")
}

func TestThePicksAreAJSONArrayOnOneLine(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, displayPicks(&out, []clips.Pick{1, 2}))

	assert.Equal(t, "[1,2]\n", out.String())
}

func TestNoPickIsAnEmptyArrayNeverNull(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, displayPicks(&out, []clips.Pick{}))

	assert.Equal(t, "[]\n", out.String())
}

func TestPicksThatCannotBeWrittenAreAnError(t *testing.T) {
	require.Error(t, displayPicks(brokenPage{}, []clips.Pick{1}))
}

func TestTheReportIsATableOfEachClipItsTimeAndWhatBecameOfIt(t *testing.T) {
	var page bytes.Buffer

	scheduled, err := displayReport(&page, clips.Report{
		{Title: "KOSOVO IS ATTACKING THE MIDDLE EAST", At: at, Post: "post-1"},
		{Title: "FRANCE IS ATTACKING EUROPE", At: at.Add(2 * time.Hour), Failure: "zernio answered 400 to POST /posts: a | b\nc"},
	})

	require.NoError(t, err)
	assert.False(t, scheduled)
	assert.Equal(t, `| Clip | Goes out | |
|---|---|---|
| KOSOVO IS ATTACKING THE MIDDLE EAST | Mon 12 Oct, 16:40 UTC | scheduled |
| FRANCE IS ATTACKING EUROPE | Mon 12 Oct, 18:40 UTC | **not scheduled**: zernio answered 400 to POST /posts: a \| b c |
`, page.String())
}

func TestAReportWithEveryClipScheduledSaysSo(t *testing.T) {
	var page bytes.Buffer

	scheduled, err := displayReport(&page, clips.Report{{Title: "T", At: at, Post: "post-1"}})

	require.NoError(t, err)
	assert.True(t, scheduled)
}

func TestATimeIsSaidInUTCWhateverItsZone(t *testing.T) {
	var page bytes.Buffer
	sofia := time.FixedZone("EEST", 3*60*60)

	_, err := displayReport(&page, clips.Report{{Title: "T", At: at.In(sofia), Post: "post-1"}})

	require.NoError(t, err)
	assert.Contains(t, page.String(), "| Mon 12 Oct, 16:40 UTC |")
}

func TestAReportThatCannotBeWrittenIsAnErrorAndNotScheduled(t *testing.T) {
	scheduled, err := displayReport(brokenPage{}, clips.Report{{Title: "T", At: at, Post: "post-1"}})

	require.Error(t, err)
	assert.False(t, scheduled)
}

func TestTheSummaryIsAddedToItsFileAndIsTheStandardOutputWhenNoFileIsNamed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "summary.md")
	require.NoError(t, os.WriteFile(path, []byte("what the workflow wrote before\n"), 0o600))
	var out bytes.Buffer

	page, closePage, err := openSummary(path, &out)
	require.NoError(t, err)
	_, err = displayReport(page, clips.Report{{Title: "T", At: at, Post: "post-1"}})
	require.NoError(t, err)
	closePage()

	written, err := os.ReadFile(path) //nolint:gosec // the file the test wrote.
	require.NoError(t, err)
	assert.Contains(t, string(written), "what the workflow wrote before\n| Clip |")
	assert.Empty(t, out.String())

	unnamed, closeUnnamed, err := openSummary("", &out)
	require.NoError(t, err)
	defer closeUnnamed()
	assert.Same(t, &out, unnamed)
}

func TestASummaryFileThatCannotBeOpenedIsAnError(t *testing.T) {
	var out bytes.Buffer

	_, _, err := openSummary(filepath.Join(t.TempDir(), "no-such-folder", "summary.md"), &out)

	require.Error(t, err)
}
