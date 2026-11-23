package fetch_flag_list_usecase_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/file_replay_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_flag_list_usecase"
)

func TestTheFlagListIsAskedOfTheBackendAndSaved(t *testing.T) {
	source := replays.NewScriptedSource(replays.Replay{})
	source.AnswerFlagList(replays.FlagList{Document: []byte(`{"flags":[{"code":"r-0000000001","name":"Bavaria","sold":{}}]}`), Sold: 1})
	out := filepath.Join(t.TempDir(), "flag-list.json")

	list, err := fetch_flag_list_usecase.New(source, file_replay_store.FlagListStore(out)).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 1, list.Sold)
	assert.Equal(t, 1, source.FlagListsAsked)
	saved, err := os.ReadFile(out) //nolint:gosec // the test's own file.
	require.NoError(t, err)
	assert.JSONEq(t, `{"flags":[{"code":"r-0000000001","name":"Bavaria","sold":{}}]}`, string(saved))
}
