package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/clips/internal/flags"
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

func TestACommandNoStageIsWrittenForIsAnErrorAndDoesNothing(t *testing.T) {
	t.Setenv(flags.KeyVariable, "the-key")
	t.Setenv(flags.HostVariable, "deploy@vps.example")
	folder := t.TempDir()

	for _, command := range []string{"", "publish", "Schedule"} {
		var out bytes.Buffer
		parsed := flags.Config{
			Command:    flags.Command(command),
			ReplayFile: filepath.Join(folder, "replay.json"), ShortlistFile: filepath.Join(folder, "shortlist.json"),
			ClipsFolder: folder, SummaryFile: filepath.Join(folder, "summary.md"),
		}

		err := runCommand(t.Context(), parsed, &out, slog.New(slog.DiscardHandler))

		require.ErrorIs(t, err, errUnknownCommand, command)
		assert.Empty(t, out.String())
	}
	left, err := os.ReadDir(folder)
	require.NoError(t, err)
	assert.Empty(t, left)
}

func TestACommandCalledWrongPrintsNothingAndSaysHowToCallIt(t *testing.T) {
	var out bytes.Buffer

	err := run(t.Context(), []string{"publish"}, &out)

	require.ErrorContains(t, err, "usage:")
	assert.Empty(t, out.String())
}

func TestAFetchOfNoTimeIsRefusedBeforeAnythingIsAskedOfTheBackend(t *testing.T) {
	t.Setenv(flags.HostVariable, "deploy@vps.example")
	var out bytes.Buffer

	err := run(t.Context(), []string{"fetch", "-out", filepath.Join(t.TempDir(), "replay.json"), "-hours", "-1"}, &out)

	require.ErrorContains(t, err, "it spans no time")
	assert.NotContains(t, err.Error(), "usage:")
}

func TestScheduleWithNoKeyFailsBeforeItSendsAnything(t *testing.T) {
	t.Setenv(flags.KeyVariable, "")
	var out bytes.Buffer

	err := run(t.Context(), []string{"schedule", "-clips", t.TempDir()}, &out)

	require.ErrorContains(t, err, "the zernio api key is empty")
	assert.Empty(t, out.String())
}

func TestScheduleWithNoClipInTheFolderFails(t *testing.T) {
	t.Setenv(flags.KeyVariable, "the-key")
	var out bytes.Buffer

	err := run(t.Context(), []string{"schedule", "-clips", t.TempDir()}, &out)

	require.ErrorContains(t, err, "no clip was rendered")
	assert.Empty(t, out.String())
}
