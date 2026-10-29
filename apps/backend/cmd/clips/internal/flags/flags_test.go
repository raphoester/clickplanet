package flags_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/clips/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

func TestFetchFlagsSayWhichReplayAndWhereItGoes(t *testing.T) {
	t.Setenv(flags.HostVariable, "")

	parsed, err := flags.Parse([]string{
		"fetch", "-ssh", "deploy@vps.example", "-out", "replay.json", "-race", "race.json", "-season", "season.json", "-cast", "cast.json", "-hours", "1.5", "-until", "2026-10-12T12:00:00Z",
	})

	require.NoError(t, err)
	var job clips.Config
	job.Replay.SSH = "deploy@vps.example"
	job.Replay.Span = 90 * time.Minute
	job.Replay.Until = time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, flags.Config{Command: flags.Fetch, ReplayFile: "replay.json", TurnFile: "season.json", RoundFile: "race.json", CastFile: "cast.json", Job: job}, parsed)
}

func TestAFetchWithNoRaceOrSeasonFileAsksForNeither(t *testing.T) {
	parsed, err := flags.Parse([]string{"fetch", "-ssh", "deploy@vps.example", "-out", "replay.json"})

	require.NoError(t, err)
	assert.Empty(t, parsed.TurnFile)
	assert.Empty(t, parsed.RoundFile)
}

func TestARaceWithNoSeasonIsRefused(t *testing.T) {
	_, err := flags.Parse([]string{"fetch", "-ssh", "deploy@vps.example", "-out", "replay.json", "-race", "race.json"})

	require.ErrorIs(t, err, flags.ErrArguments)
}

func TestFetchTakesItsHostFromTheEnvironmentAndAFlagWinsOverIt(t *testing.T) {
	t.Setenv(flags.HostVariable, "deploy@vps.example")

	fromEnvironment, err := flags.Parse([]string{"fetch", "-out", "replay.json"})
	require.NoError(t, err)
	fromFlag, err := flags.Parse([]string{"fetch", "-out", "replay.json", "-ssh", "me@other.example"})
	require.NoError(t, err)

	assert.Equal(t, "deploy@vps.example", fromEnvironment.Job.Replay.SSH)
	assert.Zero(t, fromEnvironment.Job.Replay.Span)
	assert.True(t, fromEnvironment.Job.Replay.Until.IsZero())
	assert.Equal(t, "me@other.example", fromFlag.Job.Replay.SSH)
}

func TestChooseFlagsSayWhichFindingsHowManyClipsAndTheKeyToReadWhatWasTold(t *testing.T) {
	t.Setenv(flags.KeyVariable, "the-key")
	t.Setenv(flags.EditorKeyVariable, "")

	parsed, err := flags.Parse([]string{"choose", "-findings", "findings.json", "-count", "3"})

	require.NoError(t, err)
	job := clips.Config{PerRun: 3}
	job.Zernio.APIKey = "the-key"
	assert.Equal(t, flags.Config{Command: flags.Choose, FindingsFile: "findings.json", Job: job}, parsed)
}

func TestChooseTakesTheEditorsKeyAndWhereItSaysWhoChose(t *testing.T) {
	t.Setenv(flags.KeyVariable, "the-key")
	t.Setenv(flags.EditorKeyVariable, "sk-ant-key")

	parsed, err := flags.Parse([]string{"choose", "-findings", "findings.json", "-summary", "summary.md"})

	require.NoError(t, err)
	assert.Equal(t, "sk-ant-key", parsed.Job.Anthropic.APIKey)
	assert.Equal(t, "summary.md", parsed.SummaryFile)
}

func TestEditFlagsSayWhichFindingsHowManyClipsAndWhetherOnlyThePrompt(t *testing.T) {
	t.Setenv(flags.KeyVariable, "")
	t.Setenv(flags.EditorKeyVariable, "sk-ant-key")

	parsed, err := flags.Parse([]string{"edit", "-findings", "findings.json", "-count", "3", "-prompt"})

	require.NoError(t, err)
	job := clips.Config{PerRun: 3}
	job.Anthropic.APIKey = "sk-ant-key"
	assert.Equal(t, flags.Config{Command: flags.Edit, FindingsFile: "findings.json", PromptOnly: true, Job: job}, parsed)

	_, err = flags.Parse([]string{"edit"})
	require.ErrorIs(t, err, flags.ErrArguments)
}

func TestScheduleFlagsSayWhichClipsWhenAndWhereTheSummaryGoes(t *testing.T) {
	t.Setenv(flags.KeyVariable, "the-key")

	parsed, err := flags.Parse([]string{"schedule", "-clips", "clips", "-review", "3h", "-spacing", "30m", "-summary", "summary.md"})

	require.NoError(t, err)
	var job clips.Config
	job.Review = 3 * time.Hour
	job.Spacing = 30 * time.Minute
	job.Zernio.APIKey = "the-key"
	assert.Equal(t, flags.Config{Command: flags.Schedule, ClipsFolder: "clips", SummaryFile: "summary.md", Job: job}, parsed)
}

func TestScheduleTellsTheBackendWhoThePostsNameOverTheSSHOfTheFetch(t *testing.T) {
	t.Setenv(flags.HostVariable, "deploy@vps.example")

	parsed, err := flags.Parse([]string{"schedule", "-clips", "clips"})
	require.NoError(t, err)
	assert.Equal(t, "deploy@vps.example", parsed.Job.Replay.SSH)

	parsed, err = flags.Parse([]string{"schedule", "-clips", "clips", "-ssh", "other@vps.example"})
	require.NoError(t, err)
	assert.Equal(t, "other@vps.example", parsed.Job.Replay.SSH)
}

func TestScheduleFlagsSayWhichCountrysChartATrendingTrackIsPickedFrom(t *testing.T) {
	for given, country := range map[string]string{"FR": "FR", "gb": "GB", "": "", " ": ""} {
		t.Run(given, func(t *testing.T) {
			parsed, err := flags.Parse([]string{"schedule", "-clips", "clips", "-sound-country", given})

			require.NoError(t, err)
			assert.Equal(t, country, parsed.Job.SoundCountry)
		})
	}
	for _, given := range []string{"USA", "U", "1A"} {
		_, err := flags.Parse([]string{"schedule", "-clips", "clips", "-sound-country", given})

		require.ErrorIs(t, err, clips.ErrChartCountry, given)
		assert.Contains(t, err.Error(), "usage:")
	}
}

func TestChooseCountsTheRoomOfThePlatformsItIsGiven(t *testing.T) {
	parsed, err := flags.Parse([]string{"choose", "-findings", "findings.json", "-platforms", "tiktok"})

	require.NoError(t, err)
	assert.Equal(t, []clips.Platform{"tiktok"}, parsed.Job.Platforms)
}

func TestScheduleFlagsSayWhichPlatformsEachClipGoesTo(t *testing.T) {
	for given, platforms := range map[string][]clips.Platform{
		"youtube":         {"youtube"},
		"tiktok,youtube":  {"tiktok", "youtube"},
		"youtube, tiktok": {"youtube", "tiktok"},
		"":                nil,
	} {
		t.Run(given, func(t *testing.T) {
			parsed, err := flags.Parse([]string{"schedule", "-clips", "clips", "-platforms", given})

			require.NoError(t, err)
			assert.Equal(t, platforms, parsed.Job.Platforms)
		})
	}
}

func TestFlagsWithAPartMissingOrWrongSayHowToCallTheCommand(t *testing.T) {
	t.Setenv(flags.HostVariable, "")

	for name, args := range map[string][]string{
		"nothing":                    {},
		"an unknown command":         {"publish"},
		"a command in capitals":      {"Schedule", "-clips", "clips"},
		"fetch, no file":             {"fetch", "-ssh", "deploy@vps.example"},
		"fetch, no host":             {"fetch", "-out", "replay.json"},
		"fetch, a wrong time":        {"fetch", "-ssh", "deploy@vps.example", "-out", "replay.json", "-until", "yesterday"},
		"fetch, wrong hours":         {"fetch", "-ssh", "deploy@vps.example", "-out", "replay.json", "-hours", "many"},
		"choose, no shortlist":       {"choose"},
		"choose, a wrong count":      {"choose", "-findings", "findings.json", "-count", "two"},
		"choose, no such platform":   {"choose", "-findings", "findings.json", "-platforms", "instagram"},
		"schedule, no folder":        {"schedule"},
		"schedule, a wrong gap":      {"schedule", "-clips", "clips", "-spacing", "soon"},
		"schedule, no such platform": {"schedule", "-clips", "clips", "-platforms", "tiktok,instagram"},
		"schedule, a platform twice": {"schedule", "-clips", "clips", "-platforms", "tiktok,tiktok"},
	} {
		t.Run(name, func(t *testing.T) {
			parsed, err := flags.Parse(args)

			require.ErrorContains(t, err, "usage:")
			assert.Equal(t, flags.Config{}, parsed)
		})
	}
}

func TestAMissingPartIsToldApartFromAFlagThatDoesNotParse(t *testing.T) {
	_, missing := flags.Parse([]string{"choose"})
	_, unparsed := flags.Parse([]string{"choose", "-findings", "findings.json", "-count", "two"})

	require.ErrorIs(t, missing, flags.ErrArguments)
	require.NotErrorIs(t, unparsed, flags.ErrArguments)
}

func TestStatsFlagsSayWhichBackendKeepsTheSnapshotsAndFromWhichDay(t *testing.T) {
	t.Setenv(flags.KeyVariable, "the-key")
	t.Setenv(flags.HostVariable, "")

	parsed, err := flags.Parse([]string{"stats", "-ssh", "deploy@vps.example", "-since", "2026-10-13"})

	require.NoError(t, err)
	var job clips.Config
	job.Zernio.APIKey = "the-key"
	job.Replay.SSH = "deploy@vps.example"
	job.StatsSince = time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, flags.Config{Command: flags.Stats, Job: job}, parsed)
}

func TestStatsTakeTheirHostFromTheEnvironmentAndReadTheDefaultWindowWithNoDay(t *testing.T) {
	t.Setenv(flags.HostVariable, "deploy@vps.example")

	parsed, err := flags.Parse([]string{"stats"})

	require.NoError(t, err)
	assert.Equal(t, "deploy@vps.example", parsed.Job.Replay.SSH)
	assert.True(t, parsed.Job.StatsSince.IsZero())
}

func TestStatsWithNoHostOrADayThatIsNoDayIsRefused(t *testing.T) {
	t.Setenv(flags.HostVariable, "")

	for _, args := range [][]string{
		{"stats"},
		{"stats", "-ssh", "deploy@vps.example", "-since", "13/10/2026"},
		{"stats", "-ssh", "deploy@vps.example", "-since", "2026-10-13T00:00:00Z"},
	} {
		_, err := flags.Parse(args)

		require.ErrorContains(t, err, "usage:", args)
	}
}
