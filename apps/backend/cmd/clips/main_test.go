package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shortlist(t *testing.T, written string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "shortlist.json")
	require.NoError(t, os.WriteFile(path, []byte(written), 0o600))
	return path
}

func TestChoosePrintsThePicksOfTheBestTwoStoriesAndNothingElse(t *testing.T) {
	var out bytes.Buffer
	stories := shortlist(t, `{"stories":[{"title":"BEST"},{"title":"SECOND"},{"title":"THIRD"}]}`)

	require.NoError(t, run(t.Context(), []string{"choose", "-shortlist", stories}, &out))

	assert.Equal(t, "[1,2]\n", out.String())
}

func TestChooseTakesAsManyAsItIsAskedFor(t *testing.T) {
	var out bytes.Buffer
	stories := shortlist(t, `{"stories":[{"title":"BEST"},{"title":"SECOND"},{"title":"THIRD"}]}`)

	require.NoError(t, run(t.Context(), []string{"choose", "-shortlist", stories, "-count", "1"}, &out))

	assert.Equal(t, "[1]\n", out.String())
}

func TestChoosePrintsAnEmptyListOnAQuietHalfDay(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, run(t.Context(), []string{"choose", "-shortlist", shortlist(t, "{}")}, &out))

	assert.Equal(t, "[]\n", out.String())
}

func TestACommandCalledWrongPrintsNothingAndSaysHowToCallIt(t *testing.T) {
	var out bytes.Buffer

	err := run(t.Context(), []string{"publish"}, &out)

	require.ErrorContains(t, err, "usage:")
	assert.Empty(t, out.String())
}

func TestAFetchOfNoTimeIsRefusedBeforeAnythingIsAskedOfTheBackend(t *testing.T) {
	t.Setenv(hostVariable, "deploy@vps.example")
	var out bytes.Buffer

	err := run(t.Context(), []string{"fetch", "-out", filepath.Join(t.TempDir(), "replay.json"), "-hours", "-1"}, &out)

	require.ErrorContains(t, err, "it spans no time")
	assert.NotContains(t, err.Error(), "usage:")
}

func TestScheduleWithNoKeyFailsBeforeItSendsAnything(t *testing.T) {
	t.Setenv(keyVariable, "")
	var out bytes.Buffer

	err := run(t.Context(), []string{"schedule", "-clips", t.TempDir()}, &out)

	require.ErrorContains(t, err, "the zernio api key is empty")
	assert.Empty(t, out.String())
}

func TestScheduleWithNoClipInTheFolderFails(t *testing.T) {
	t.Setenv(keyVariable, "the-key")
	var out bytes.Buffer

	err := run(t.Context(), []string{"schedule", "-clips", t.TempDir()}, &out)

	require.ErrorContains(t, err, "no clip was rendered")
	assert.Empty(t, out.String())
}
