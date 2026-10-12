package markdown_summary_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/markdown_summary"
)

var at = time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC)

func TestTheSummaryListsEachClipWithItsTimeAndWhatBecameOfIt(t *testing.T) {
	var page bytes.Buffer

	err := markdown_summary.New(&page).Write(posts.Report{
		{Title: "KOSOVO IS ATTACKING THE MIDDLE EAST", At: at, Post: "post-1"},
		{Title: "FRANCE IS ATTACKING EUROPE", At: at.Add(2 * time.Hour), Failure: "zernio answered 400 to POST /posts: a | b\nc"},
	})

	require.NoError(t, err)
	assert.Equal(t, `| Clip | Goes out | |
|---|---|---|
| KOSOVO IS ATTACKING THE MIDDLE EAST | Mon 12 Oct, 16:40 UTC | scheduled |
| FRANCE IS ATTACKING EUROPE | Mon 12 Oct, 18:40 UTC | **not scheduled**: zernio answered 400 to POST /posts: a \| b c |
`, page.String())
}

func TestATimeIsSaidInUTCWhateverItsZone(t *testing.T) {
	var page bytes.Buffer
	sofia := time.FixedZone("EEST", 3*60*60)

	require.NoError(t, markdown_summary.New(&page).Write(posts.Report{{Title: "T", At: at.In(sofia), Post: "post-1"}}))

	assert.Contains(t, page.String(), "| Mon 12 Oct, 16:40 UTC |")
}

type brokenPage struct{}

func (brokenPage) Write([]byte) (int, error) {
	return 0, errors.New("the disk is full")
}

func TestAPageThatCannotBeWrittenIsAnError(t *testing.T) {
	require.Error(t, markdown_summary.New(brokenPage{}).Write(posts.Report{{Title: "T", At: at, Post: "post-1"}}))
}
