package flags_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/missionsim/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions"
)

func TestFetchNeedsAHostAStartAndAFolder(t *testing.T) {
	t.Setenv(flags.HostVariable, "")

	parsed, err := flags.Parse([]string{"fetch", "-ssh", "root@vps", "-since", "2026-11-01T06:00:00Z", "-out", "season"})

	require.NoError(t, err)
	assert.Equal(t, flags.Config{
		Command: flags.Fetch, Host: "root@vps", Since: time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC), Folder: "season",
	}, parsed)
	for _, args := range [][]string{
		{"fetch", "-since", "2026-11-01T06:00:00Z", "-out", "season"},
		{"fetch", "-ssh", "root@vps", "-out", "season"},
		{"fetch", "-ssh", "root@vps", "-since", "2026-11-01T06:00:00Z"},
	} {
		_, err := flags.Parse(args)
		assert.ErrorIs(t, err, flags.ErrArguments, args)
	}
}

func TestTheHostCanComeFromTheEnvironment(t *testing.T) {
	t.Setenv(flags.HostVariable, "root@vps")

	parsed, err := flags.Parse([]string{"fetch", "-since", "2026-11-01T06:00:00Z", "-out", "season"})

	require.NoError(t, err)
	assert.Equal(t, "root@vps", parsed.Host)
}

func TestRunReadsTheSizingsToTry(t *testing.T) {
	sizings := filepath.Join(t.TempDir(), "sizings.json")
	require.NoError(t, os.WriteFile(sizings, []byte(`[{"name": "harder", "goal_percent": 80, "least_conquest": 300,
		"least_shields": 300, "fewest_points": 2, "most_points": 6}]`), 0o600))

	parsed, err := flags.Parse([]string{"run", "-data", "season", "-sizings", sizings, "-rows", "rows.csv"})

	require.NoError(t, err)
	assert.Equal(t, flags.Config{Command: flags.Run, Folder: "season", Rows: "rows.csv", Sizings: []missions.NamedSizing{{
		Name:   "harder",
		Sizing: missions.Sizing{GoalPercent: 80, LeastConquest: 300, LeastShields: 300, FewestPoints: 2, MostPoints: 6},
	}}}, parsed)
}

func TestASizingNeedsANameOfItsOwnAndKnownFields(t *testing.T) {
	for _, document := range []string{
		`[{"goal_percent": 80, "fewest_points": 2, "most_points": 6}]`,
		`[{"name": "played", "goal_percent": 80, "fewest_points": 2, "most_points": 6}]`,
		`[{"name": "harder", "goal_pct": 80}]`,
	} {
		sizings := filepath.Join(t.TempDir(), "sizings.json")
		require.NoError(t, os.WriteFile(sizings, []byte(document), 0o600))

		_, err := flags.Parse([]string{"run", "-data", "season", "-sizings", sizings})

		assert.ErrorIs(t, err, flags.ErrArguments, document)
	}
}

func TestAnUnknownCommandIsRefused(t *testing.T) {
	for _, args := range [][]string{nil, {"post"}, {"run"}} {
		_, err := flags.Parse(args)

		assert.ErrorIs(t, err, flags.ErrArguments, args)
	}
}
