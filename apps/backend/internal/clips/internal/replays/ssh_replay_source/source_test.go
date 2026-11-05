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

const turned = `{"turn":{"kind":"ENTRY_KIND_PLAIN"},` +
	`"timeline":{"current":{"number":0,"bookId":"season-0"},"battles":[` +
	`{"entryId":"a","startsAt":"2026-10-29T18:00:00Z","times":2},` +
	`{"entryId":"b","startsAt":"2026-10-31T21:00:00Z","endsSeason":true,"roundId":"b"}]},` +
	`"debrief":{"day":{"season":0,"day":20,"roundId":"season-0/day-20","endedAt":"2026-10-27T21:00:00Z"}},` +
	`"aFieldOfALaterBackend":true}`

func TestTheTurnIsAskedOfThePublicListenerOverSSH(t *testing.T) {
	fake := ssh(t, turned, "", 0)
	source, err := ssh_replay_source.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	turn, err := source.Turn(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []string{
		"--",
		"deploy@vps.example",
		"cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO- --header 'Content-Type: application/json'" +
			" --post-data '{}' http://127.0.0.1:8080/gamemaster.v1.GamemasterService/GetTurn",
	}, fake.asked(t))
	assert.Equal(t, replays.Turn{
		Document:  []byte(turned),
		Debriefed: "0.20",
		Round:     "season-0/day-20",
		EndedAt:   time.Date(2026, 10, 27, 21, 0, 0, 0, time.UTC),
		FinaleAt:  time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
	}, turn)
}

func TestABattleDebriefedIsLabelledApartFromTheDayOfItsNumber(t *testing.T) {
	battle := `{"debrief":{"battle":{"season":0,"number":1,"times":3,"roundId":"5d3a1c2e-1b7f-4e0c-8a9d-2f6b4c8e0a13"}}}`
	source, err := ssh_replay_source.New("deploy@vps.example", ssh(t, battle, "", 0).client)
	require.NoError(t, err)

	turn, err := source.Turn(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "0.battle.1", turn.Debriefed)
	assert.Equal(t, replays.RoundID("5d3a1c2e-1b7f-4e0c-8a9d-2f6b4c8e0a13"), turn.Round)
}

func TestASeasonDebriefedIsItsFinaleBattleByItsNumber(t *testing.T) {
	season := `{"debrief":{"season":{"season":0,"bookId":"season-0","finaleRoundId":"b","endedAt":"2026-10-31T23:00:00Z",` +
		`"finaleBattle":{"season":0,"number":3,"times":3,"endsSeason":true,"roundId":"b"}}}}`
	source, err := ssh_replay_source.New("deploy@vps.example", ssh(t, season, "", 0).client)
	require.NoError(t, err)

	turn, err := source.Turn(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "0.battle.3", turn.Debriefed)
	assert.Equal(t, replays.RoundID("b"), turn.Round)
}

func TestASeasonDebriefedWithNoFinaleBattleIsItsFinalesRound(t *testing.T) {
	season := `{"debrief":{"season":{"season":0,"bookId":"season-0","finaleRoundId":"b","endedAt":"2026-10-31T23:00:00Z"}}}`
	source, err := ssh_replay_source.New("deploy@vps.example", ssh(t, season, "", 0).client)
	require.NoError(t, err)

	turn, err := source.Turn(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "0.finale", turn.Debriefed)
	assert.Equal(t, replays.RoundID("b"), turn.Round)
	assert.Equal(t, time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), turn.EndedAt)
}

func TestATurnWithNothingScoredYetIsATurnToo(t *testing.T) {
	source, err := ssh_replay_source.New("deploy@vps.example", ssh(t, `{"turn":{}}`, "", 0).client)
	require.NoError(t, err)

	turn, err := source.Turn(t.Context())

	require.NoError(t, err)
	assert.Equal(t, replays.Turn{Document: []byte(`{"turn":{}}`)}, turn)
}

func TestADebriefedRoundNoShellCanBeTrustedWithIsAnError(t *testing.T) {
	source, err := ssh_replay_source.New("deploy@vps.example",
		ssh(t, `{"debrief":{"day":{"season":0,"day":1,"roundId":"x'; rm -rf /"}}}`, "", 0).client)
	require.NoError(t, err)

	_, err = source.Turn(t.Context())

	require.ErrorIs(t, err, replays.ErrInvalidRoundID)
}

func TestATurnTheBackendDoesNotServeIsAnErrorThatSaysWhy(t *testing.T) {
	source, err := ssh_replay_source.New("deploy@vps.example", ssh(t, "", "wget: server returned error: HTTP/1.1 404 Not Found\n", 1).client)
	require.NoError(t, err)

	_, err = source.Turn(t.Context())

	require.ErrorContains(t, err, "ssh failed: exit status 1: wget: server returned error: HTTP/1.1 404 Not Found")
}

func TestAnAnswerThatIsNotATurnIsAnError(t *testing.T) {
	for name, says := range map[string]string{
		"nothing": "",
		"a page":  "<html>502 Bad Gateway</html>",
		"a list":  `[]`,
	} {
		t.Run(name, func(t *testing.T) {
			source, err := ssh_replay_source.New("deploy@vps.example", ssh(t, says, "", 0).client)
			require.NoError(t, err)

			_, err = source.Turn(t.Context())

			require.Error(t, err)
		})
	}
}

const rounded = `{"round":{"roundId":"season-0/day-20","bookId":"season-0","times":1,"endedAtUnixMs":"1793048400000",` +
	`"standings":[{"rank":1,"countryId":"fr","share":0.18,"points":25}],"after":[{"rank":1,"countryId":"fr","points":86}]},` +
	`"aFieldOfALaterBackend":true}`

func TestTheRoundIsAskedOfThePublicListenerByItsID(t *testing.T) {
	fake := ssh(t, rounded, "", 0)
	source, err := ssh_replay_source.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	round, err := source.Round(t.Context(), "season-0/day-20")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"--",
		"deploy@vps.example",
		"cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO- --header 'Content-Type: application/json'" +
			` --post-data '{"roundId":"season-0/day-20"}' http://127.0.0.1:8080/scoring.v1.ScoringService/GetRound`,
	}, fake.asked(t))
	assert.Equal(t, replays.Round{Document: []byte(rounded), ID: "season-0/day-20"}, round)
}

func TestAnAnswerThatIsNotTheRoundAskedIsAnError(t *testing.T) {
	for name, says := range map[string]string{
		"nothing":       "",
		"a page":        "<html>502 Bad Gateway</html>",
		"no round":      `{}`,
		"another round": `{"round":{"roundId":"season-0/day-19"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			source, err := ssh_replay_source.New("deploy@vps.example", ssh(t, says, "", 0).client)
			require.NoError(t, err)

			_, err = source.Round(t.Context(), "season-0/day-20")

			require.Error(t, err)
		})
	}
}

func TestTheCastIsAskedOfTheOperatorToolAndWrittenAlone(t *testing.T) {
	fake := ssh(t, `{"cast":{"id":"5f0c2a11-0b6d-4f7e-9a51-3f1f8f0c2a11","fighters":[{"number":1,"name":"BraveFox42"}],`+
		`"takes":[{"tileId":7,"countryId":"fr","previousCountryId":"de","fighter":1}]},"aFieldOfALaterBackend":true}`, "", 0)
	source, err := ssh_replay_source.New("deploy@vps.example", fake.client)
	require.NoError(t, err)

	cast, err := source.Cast(t.Context(), window)

	require.NoError(t, err)
	assert.Equal(t, "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO- --header 'Content-Type: application/json'"+
		` --post-data '{"since":"2026-10-12T00:17:00Z","until":"2026-10-12T12:17:00.5Z"}'`+
		" http://127.0.0.1:8081/clips.v1.AdminService/MakeCast", fake.asked(t)[2])
	assert.JSONEq(t, `{"id":"5f0c2a11-0b6d-4f7e-9a51-3f1f8f0c2a11","fighters":[{"number":1,"name":"BraveFox42"}],`+
		`"takes":[{"tileId":7,"countryId":"fr","previousCountryId":"de","fighter":1}]}`, string(cast.Document))
	assert.Equal(t, 1, cast.Fighters)
	assert.Equal(t, 1, cast.Takes)
}

func TestACastWithNoIDIsAnError(t *testing.T) {
	source, err := ssh_replay_source.New("deploy@vps.example", ssh(t, `{}`, "", 0).client)
	require.NoError(t, err)

	_, err = source.Cast(t.Context(), window)

	assert.ErrorContains(t, err, "no id")
}
