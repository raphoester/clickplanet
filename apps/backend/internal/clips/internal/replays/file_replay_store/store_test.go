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

func TestATurnIsWrittenAsTheBackendAnsweredIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "season.json")

	require.NoError(t, file_replay_store.TurnStore(path).SaveTurn(t.Context(), replays.Turn{Document: []byte(`{"timeline":{}}`)}))

	written, err := os.ReadFile(path) //nolint:gosec // the file the test wrote.
	require.NoError(t, err)
	assert.JSONEq(t, `{"timeline":{}}`, string(written))
}

func TestATurnThatCannotBeWrittenIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-folder", "season.json")

	require.Error(t, file_replay_store.TurnStore(path).SaveTurn(t.Context(), replays.Turn{Document: []byte(`{}`)}))
}

func TestARoundIsWrittenAsTheBackendAnsweredIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "race.json")

	require.NoError(t, file_replay_store.RoundStore(path).SaveRound(t.Context(), replays.Round{Document: []byte(`{"round":{}}`)}))

	written, err := os.ReadFile(path) //nolint:gosec // the file the test wrote.
	require.NoError(t, err)
	assert.JSONEq(t, `{"round":{}}`, string(written))
}

func TestARoundThatCannotBeWrittenIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-folder", "race.json")

	require.Error(t, file_replay_store.RoundStore(path).SaveRound(t.Context(), replays.NoRound))
}
