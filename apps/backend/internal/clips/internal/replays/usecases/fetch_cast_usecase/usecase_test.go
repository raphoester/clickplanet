package fetch_cast_usecase_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/file_replay_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_cast_usecase"
)

var window = replays.Window{Since: time.Date(2026, 10, 28, 0, 17, 0, 0, time.UTC), Until: time.Date(2026, 10, 28, 12, 17, 0, 0, time.UTC)}

func TestTheCastOfTheReplaysWindowIsSaved(t *testing.T) {
	source := replays.NewScriptedSource(replays.Replay{})
	source.AnswerCast(replays.Cast{Document: []byte(`{"id":"c"}`), Fighters: 1})
	out := filepath.Join(t.TempDir(), "cast.json")

	cast, err := fetch_cast_usecase.New(source, file_replay_store.CastStore(out)).Execute(t.Context(), window)

	require.NoError(t, err)
	assert.Equal(t, 1, cast.Fighters)
	assert.Equal(t, []replays.Window{window}, source.CastsAsked)
	saved, err := os.ReadFile(out) //nolint:gosec // the test's own file.
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"c"}`, string(saved))
}

func TestACastTheBackendDoesNotAnswerSavesNothing(t *testing.T) {
	source := replays.NewScriptedSource(replays.Replay{})
	source.FailCastWith(errors.New("no route to host"))
	out := filepath.Join(t.TempDir(), "cast.json")

	_, err := fetch_cast_usecase.New(source, file_replay_store.CastStore(out)).Execute(t.Context(), window)

	require.ErrorContains(t, err, "no route to host")
	assert.NoFileExists(t, out)
}
