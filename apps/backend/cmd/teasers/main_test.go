package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo"
)

func TestACommandCalledWrongSaysHowToCallItAndAsksNothing(t *testing.T) {
	t.Setenv(hostVariable, "")
	folder := t.TempDir()

	for _, args := range [][]string{
		nil,
		{"publish"},
		{"due"},
		{"due", "-out", folder},
		{"post", "-teasers", folder},
		{"post", "-ssh", "deploy@vps.example"},
		{"due", "-out", folder, "-ssh", "deploy@vps.example", "-hours", "3"},
	} {
		var out bytes.Buffer

		err := run(t.Context(), args, &out)

		require.ErrorIs(t, err, errArguments, args)
		require.ErrorContains(t, err, "usage:")
		assert.Empty(t, out.String())
	}
	left, err := os.ReadDir(folder)
	require.NoError(t, err)
	assert.Empty(t, left)
}

// An ssh on the PATH that answers every call with nothing: no battle, no post.
func quietSSH(t *testing.T) {
	t.Helper()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(folder, "ssh"), []byte("#!/bin/sh\ncat > /dev/null\necho '{}'\n"), 0o700)) //nolint:gosec // a script the test runs.
	t.Setenv("PATH", folder+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestNoBattleDueIsAnEmptyMatrix(t *testing.T) {
	quietSSH(t)
	t.Setenv(hostVariable, "deploy@vps.example")
	var out bytes.Buffer

	require.NoError(t, run(t.Context(), []string{"due", "-out", t.TempDir()}, &out))

	assert.Equal(t, "[]\n", out.String())
}

func TestNothingToPostIsSaidInTheSummary(t *testing.T) {
	quietSSH(t)
	summary := filepath.Join(t.TempDir(), "summary.md")
	var out bytes.Buffer

	require.NoError(t, run(t.Context(), []string{"post", "-teasers", t.TempDir(), "-ssh", "deploy@vps.example", "-summary", summary}, &out))

	written, err := os.ReadFile(summary) //nolint:gosec // a file the test made.
	require.NoError(t, err)
	assert.Equal(t, "## Battle teasers\n\nNo teaser to post this run.\n", string(written))
	assert.Empty(t, out.String())
}

func TestTheSummarySaysWhatBecameOfEachTeaser(t *testing.T) {
	at := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)

	summary := summaryOf(promo.TeasersReport{
		{
			Battle: "entry-7", Title: "BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC", Result: promo.TeaserPosted, At: at,
			Post: "post-1", Platforms: []string{"tiktok", "youtube"}, Refused: map[string]string{"youtube": "no account"},
		},
		{Battle: "season-1/finale", Result: promo.TeaserHeld, Why: "no platform takes it"},
	})

	assert.Equal(t, "## Battle teasers\n\n"+
		"- **BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC**: posted at Fri 13 Nov 14:00 UTC on tiktok and youtube (post post-1); youtube refused it: no account\n"+
		"- **season-1/finale**: held, no platform takes it\n", summary)
}
