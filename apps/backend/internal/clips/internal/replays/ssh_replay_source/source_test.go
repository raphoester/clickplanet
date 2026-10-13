package ssh_replay_source_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/ssh_replay_source"
)

const answered = `{"opening":{"startTileId":1,"codes":["","fr"],"tiles":"AQA="},` +
	`"events":[{"at":"2026-10-12T00:20:00Z","event":{}},{"at":"2026-10-12T00:21:00Z","event":{}},{"at":"2026-10-12T00:22:00Z","event":{}}],` +
	`"since":"2026-10-12T00:17:00Z","until":"2026-10-12T12:17:00Z","aFieldOfALaterBackend":true}`

var window = replays.Window{
	Since: time.Date(2026, 10, 12, 0, 17, 0, 0, time.UTC),
	Until: time.Date(2026, 10, 12, 12, 17, 0, 500_000_000, time.UTC),
}

type fakeSSH struct {
	client string
	args   string
}

func (f fakeSSH) asked(t *testing.T) []string {
	t.Helper()

	raw, err := os.ReadFile(f.args)
	require.NoError(t, err)
	return strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
}

// An ssh that keeps its arguments, one a line, then says and exits as told.
func ssh(t *testing.T, says, complains string, exit int) fakeSSH {
	t.Helper()

	folder := t.TempDir()
	fake := fakeSSH{client: filepath.Join(folder, "ssh"), args: filepath.Join(folder, "args")}
	require.NoError(t, os.WriteFile(filepath.Join(folder, "says"), []byte(says), 0o600))
	script := fmt.Sprintf("#!/bin/sh\nfor arg in \"$@\"; do printf '%%s\\n' \"$arg\"; done > %q\ncat %q\nprintf '%%s' %q >&2\nexit %d\n",
		fake.args, filepath.Join(folder, "says"), complains, exit)
	require.NoError(t, os.WriteFile(fake.client, []byte(script), 0o700)) //nolint:gosec // a script the test runs.
	return fake
}

func TestAReplayIsAskedOfTheOperatorToolOverSSH(t *testing.T) {
	fake := ssh(t, answered, "", 0)
	source, err := ssh_replay_source.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	replay, err := source.Replay(t.Context(), window)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"--",
		"deploy@vps.example",
		"cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO- --header 'Content-Type: application/json'" +
			` --post-data '{"since":"2026-10-12T00:17:00Z","until":"2026-10-12T12:17:00.5Z"}'` +
			" http://127.0.0.1:8081/planet.v1.AdminService/GetReplay",
	}, fake.asked(t))
	assert.Equal(t, replays.Replay{
		Document: []byte(answered),
		Events:   3,
		Window:   replays.Window{Since: window.Since, Until: time.Date(2026, 10, 12, 12, 17, 0, 0, time.UTC)},
	}, replay)
}

func TestAHostThatStartsWithADashIsNeverAnOptionOfSSH(t *testing.T) {
	fake := ssh(t, answered, "", 0)
	source, err := ssh_replay_source.New("-oProxyCommand=evil", fake.client)
	require.NoError(t, err)

	_, err = source.Replay(t.Context(), window)

	require.NoError(t, err)
	assert.Equal(t, []string{"--", "-oProxyCommand=evil"}, fake.asked(t)[:2])
}

func TestAnSSHThatFailsIsAnErrorThatSaysWhatItComplainedOf(t *testing.T) {
	fake := ssh(t, "", "Permission denied (publickey).\n", 255)
	source, err := ssh_replay_source.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	_, err = source.Replay(t.Context(), window)

	require.ErrorContains(t, err, "ssh failed: exit status 255: Permission denied (publickey).")
}

func TestAnAnswerThatIsNotAReplayIsAnError(t *testing.T) {
	for name, says := range map[string]string{
		"nothing":            "",
		"a page":             "<html>502 Bad Gateway</html>",
		"JSON with no times": `{"events":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			source, err := ssh_replay_source.New("deploy@vps.example", ssh(t, says, "", 0).client)
			require.NoError(t, err)

			_, err = source.Replay(t.Context(), window)

			require.Error(t, err)
		})
	}
}

func TestASourceWithNoHostIsRefusedWhenItIsBuilt(t *testing.T) {
	_, err := ssh_replay_source.New("", ssh_replay_source.OpenSSH)

	require.Error(t, err)
}
