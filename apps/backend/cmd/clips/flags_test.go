package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

func TestFetchFlagsSayWhichReplayAndWhereItGoes(t *testing.T) {
	t.Setenv(hostVariable, "")

	flags, err := parseFlags([]string{
		"fetch", "-ssh", "deploy@vps.example", "-out", "replay.json", "-hours", "1.5", "-until", "2026-10-12T12:00:00Z",
	})

	require.NoError(t, err)
	var job clips.Config
	job.Replay.SSH = "deploy@vps.example"
	job.Replay.Span = 90 * time.Minute
	job.Replay.Until = time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, FlagsConfig{Command: fetchCommand, ReplayFile: "replay.json", Job: job}, flags)
}

func TestFetchTakesItsHostFromTheEnvironmentAndAFlagWinsOverIt(t *testing.T) {
	t.Setenv(hostVariable, "deploy@vps.example")

	fromEnvironment, err := parseFlags([]string{"fetch", "-out", "replay.json"})
	require.NoError(t, err)
	fromFlag, err := parseFlags([]string{"fetch", "-out", "replay.json", "-ssh", "me@other.example"})
	require.NoError(t, err)

	assert.Equal(t, "deploy@vps.example", fromEnvironment.Job.Replay.SSH)
	assert.Zero(t, fromEnvironment.Job.Replay.Span)
	assert.True(t, fromEnvironment.Job.Replay.Until.IsZero())
	assert.Equal(t, "me@other.example", fromFlag.Job.Replay.SSH)
}

func TestChooseFlagsSayWhichShortlistAndHowManyClips(t *testing.T) {
	flags, err := parseFlags([]string{"choose", "-shortlist", "shortlist.json", "-count", "3"})

	require.NoError(t, err)
	assert.Equal(t, FlagsConfig{Command: chooseCommand, ShortlistFile: "shortlist.json", Job: clips.Config{PerRun: 3}}, flags)
}

func TestScheduleFlagsSayWhichClipsWhenAndWhereTheSummaryGoes(t *testing.T) {
	t.Setenv(keyVariable, "the-key")

	flags, err := parseFlags([]string{"schedule", "-clips", "clips", "-review", "3h", "-spacing", "30m", "-summary", "summary.md"})

	require.NoError(t, err)
	var job clips.Config
	job.Review = 3 * time.Hour
	job.Spacing = 30 * time.Minute
	job.Zernio.APIKey = "the-key"
	assert.Equal(t, FlagsConfig{Command: scheduleCommand, ClipsFolder: "clips", SummaryFile: "summary.md", Job: job}, flags)
}

func TestFlagsWithAPartMissingOrWrongSayHowToCallTheCommand(t *testing.T) {
	t.Setenv(hostVariable, "")

	for name, args := range map[string][]string{
		"nothing":               {},
		"an unknown command":    {"publish"},
		"fetch, no file":        {"fetch", "-ssh", "deploy@vps.example"},
		"fetch, no host":        {"fetch", "-out", "replay.json"},
		"fetch, a wrong time":   {"fetch", "-ssh", "deploy@vps.example", "-out", "replay.json", "-until", "yesterday"},
		"fetch, wrong hours":    {"fetch", "-ssh", "deploy@vps.example", "-out", "replay.json", "-hours", "many"},
		"choose, no shortlist":  {"choose"},
		"choose, a wrong count": {"choose", "-shortlist", "shortlist.json", "-count", "two"},
		"schedule, no folder":   {"schedule"},
		"schedule, a wrong gap": {"schedule", "-clips", "clips", "-spacing", "soon"},
	} {
		t.Run(name, func(t *testing.T) {
			flags, err := parseFlags(args)

			require.ErrorContains(t, err, "usage:")
			assert.Equal(t, FlagsConfig{}, flags)
		})
	}
}
