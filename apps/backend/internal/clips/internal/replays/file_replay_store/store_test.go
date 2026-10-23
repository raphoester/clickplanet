package file_replay_store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/file_replay_store"
)

func TestAReplayIsWrittenAsTheBackendAnsweredIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "replay.json")

	require.NoError(t, file_replay_store.Store(path).Save(t.Context(), replays.Replay{Document: []byte(`{"events":[]}`)}))

	written, err := os.ReadFile(path) //nolint:gosec // the file the test wrote.
	require.NoError(t, err)
	assert.JSONEq(t, `{"events":[]}`, string(written))
}

func TestAReplayThatCannotBeWrittenIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-folder", "replay.json")

	require.Error(t, file_replay_store.Store(path).Save(t.Context(), replays.Replay{Document: []byte(`{}`)}))
}

func TestARaceIsWrittenAsTheBackendAnsweredIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "race.json")

	require.NoError(t, file_replay_store.RaceStore(path).SaveRace(t.Context(), replays.Race{Document: []byte(`{"race":{}}`)}))

	written, err := os.ReadFile(path) //nolint:gosec // the file the test wrote.
	require.NoError(t, err)
	assert.JSONEq(t, `{"race":{}}`, string(written))
}

func TestARaceThatCannotBeWrittenIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-folder", "race.json")

	require.Error(t, file_replay_store.RaceStore(path).SaveRace(t.Context(), replays.Race{Document: []byte(`{}`)}))
}

func TestASeasonIsWrittenAsTheBackendAnsweredIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "season.json")

	require.NoError(t, file_replay_store.SeasonStore(path).SaveSeason(t.Context(), replays.Season{Document: []byte(`{"season":{}}`)}))

	written, err := os.ReadFile(path) //nolint:gosec // the file the test wrote.
	require.NoError(t, err)
	assert.JSONEq(t, `{"season":{}}`, string(written))
}

func TestASeasonThatCannotBeWrittenIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-folder", "season.json")

	require.Error(t, file_replay_store.SeasonStore(path).SaveSeason(t.Context(), replays.Season{Document: []byte(`{}`)}))
}
