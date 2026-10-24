package display_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/clips/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

var snapshotAt = time.Date(2026, 10, 24, 5, 41, 0, 0, time.UTC)

func aReport() clips.StatsReport {
	return clips.StatsReport{
		TakenAt: snapshotAt, Since: time.Date(2026, 9, 26, 5, 41, 0, 0, time.UTC),
		Posts: 40, NotOut: 3, Rows: map[string]int{"tiktok": 37, "youtube": 36}, Syncing: 4,
		Curves: 30, CurvesRefused: map[string]int{"empty": 6},
		TikTokAccounts: []clips.TikTokLane{{
			TikTokAccount: clips.TikTokAccount{
				ID: "tt-1", Username: "clickplanet", Type: "personal",
				CommercialMusicRefusal: "zernio answered 400 to the commercial music library: Not a business account",
			},
			Rows: 37,
		}},
	}
}

func TestTheSummaryOfASnapshotSaysWhatWasKeptAndWhetherTikToksBusinessLaneIsLive(t *testing.T) {
	var out bytes.Buffer

	whole, err := display.Stats("", &out, aReport())

	require.NoError(t, err)
	assert.True(t, whole)
	assert.Equal(t, "### Snapshot of Sat 24 Oct, 05:41 UTC\n\n"+
		"40 posts of the job since Sat 26 Sep 2026, 3 of them not out yet. 73 rows kept (37 tiktok, 36 youtube), 4 still syncing.\n\n"+
		"| TikTok account | tiktokAccountType | Rows with business metrics | Commercial music | Business lane |\n"+
		"|---|---|---|---|---|\n"+
		"| @clickplanet | personal | 0 of 37 | refused: zernio answered 400 to the commercial music library: Not a business account | not live |\n\n"+
		"TikTok sends its business metrics 24 to 48 hours after a post, and only for posts active in the last 7 days.\n\n"+
		"YouTube retention curves: 30, none for 6 empty.\n\n", out.String())
}

func TestALiveLaneAndAMissingScopeAreSaidPlainly(t *testing.T) {
	report := aReport()
	report.TikTokAccounts[0].Type, report.TikTokAccounts[0].CommercialMusic, report.TikTokAccounts[0].BusinessRows = "business", true, 12
	report.CurvesRefused = map[string]int{"412 youtube_analytics_scope_missing": 36}
	var out bytes.Buffer

	_, err := display.Stats("", &out, report)

	require.NoError(t, err)
	assert.Contains(t, out.String(), "| @clickplanet | business | 12 of 37 | answers | **live** |")
	assert.Contains(t, out.String(), "**YouTube's analytics scope is missing**: reconnect the YouTube account in Zernio to grant it.")
}

func TestAPostNotReadWholeIsListedAndTheSnapshotIsNotWhole(t *testing.T) {
	report := aReport()
	report.Unread = []clips.Unread{{PostID: "post-9", Reason: "zernio answered 404 | gone"}}
	summary := filepath.Join(t.TempDir(), "summary.md")
	require.NoError(t, os.WriteFile(summary, []byte("before\n"), 0o600))

	whole, err := display.Stats(summary, &bytes.Buffer{}, report)

	require.NoError(t, err)
	assert.False(t, whole)
	written, err := os.ReadFile(summary) //nolint:gosec // the file the test wrote.
	require.NoError(t, err)
	assert.Contains(t, string(written), "before\n### Snapshot of")
	assert.Contains(t, string(written), "| post-9 | zernio answered 404 \\| gone |")
}

func TestNoTikTokAccountIsSaid(t *testing.T) {
	report := aReport()
	report.TikTokAccounts = nil
	var out bytes.Buffer

	_, err := display.Stats("", &out, report)

	require.NoError(t, err)
	assert.Contains(t, out.String(), "No TikTok account is connected.")
}

func TestASummaryThatCannotBeWrittenIsAnError(t *testing.T) {
	_, err := display.Stats("", brokenPage{}, aReport())

	require.ErrorContains(t, err, "the disk is full")
}
