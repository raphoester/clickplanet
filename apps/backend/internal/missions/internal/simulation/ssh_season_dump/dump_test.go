package ssh_season_dump_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/simulation/csv_season"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/simulation/ssh_season_dump"
)

var since = time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)

type fakeSSH struct {
	client string
	folder string
}

// The arguments and the standard input of each call, one file each, numbered.
func (f fakeSSH) call(t *testing.T, n int) (args, input string) {
	t.Helper()

	rawArgs, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("args-%d", n)))
	require.NoError(t, err)
	rawInput, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("input-%d", n)))
	require.NoError(t, err)
	return string(rawArgs), string(rawInput)
}

// An ssh that keeps its arguments and its input, then says what it was told for that call and exits as told.
func ssh(t *testing.T, says []string, complains string, exit int) fakeSSH {
	t.Helper()

	folder := t.TempDir()
	for n, said := range says {
		require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("says-%d", n+1)), []byte(said), 0o600))
	}
	script := fmt.Sprintf("#!/bin/sh\nn=$(($(ls %[1]q | grep -c '^args-') + 1))\n"+
		"printf '%%s\\n' \"$@\" > %[1]q/args-$n\ncat > %[1]q/input-$n\ncat %[1]q/says-$n 2>/dev/null\n"+
		"printf '%%s' %[2]q >&2\nexit %[3]d\n", folder, complains, exit)
	client := filepath.Join(t.TempDir(), "ssh")
	require.NoError(t, os.WriteFile(client, []byte(script), 0o700)) //nolint:gosec // a script the test runs.
	return fakeSSH{client: client, folder: folder}
}

func TestTheDaysAndTheActsAreCopiedFromPostgresIntoTheFolder(t *testing.T) {
	fake := ssh(t, []string{"season-1/day-1,1,1,1,2\n", "4,0,3,take,se,fr,4,t\n"}, "", 0)
	dump, err := ssh_season_dump.New("deploy@vps.example", fake.client)
	require.NoError(t, err)
	out := filepath.Join(t.TempDir(), "season")

	require.NoError(t, dump.Write(t.Context(), since, out))

	for n, file := range []string{csv_season.DaysFile, csv_season.ActsFile} {
		args, input := fake.call(t, n+1)
		assert.Equal(t, "--\ndeploy@vps.example\n"+
			"docker exec -i cp-postgres psql -U clickplanet -d clickplanet -X -q -v ON_ERROR_STOP=1\n", args)
		assert.Contains(t, input, "'2026-11-01T06:00:00Z'::timestamptz", "the start is in the query, never on the command line")
		assert.True(t, strings.HasPrefix(input, "COPY ("), input)
		written, err := os.ReadFile(filepath.Join(out, file)) //nolint:gosec // the test's own temporary folder.
		require.NoError(t, err)
		assert.Equal(t, []string{"season-1/day-1,1,1,1,2\n", "4,0,3,take,se,fr,4,t\n"}[n], string(written))
	}
	_, daysQuery := fake.call(t, 1)
	assert.Contains(t, daysQuery, "missions.mission_days")
	_, actsQuery := fake.call(t, 2)
	assert.Contains(t, actsQuery, "planet.ledger_events")
}

func TestWhatSSHComplainsAboutIsTheError(t *testing.T) {
	fake := ssh(t, nil, "permission denied", 255)
	dump, err := ssh_season_dump.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	err = dump.Write(t.Context(), since, t.TempDir())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestAHostIsNeeded(t *testing.T) {
	_, err := ssh_season_dump.New("", ssh_season_dump.OpenSSH)

	assert.Error(t, err)
}
