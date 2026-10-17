package log_choose_clips_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase/log_choose_clips"
)

type stubExecutor struct {
	choice posts.Choice
	err    error
}

func (s stubExecutor) Execute(context.Context) (posts.Choice, error) {
	return s.choice, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_choose_clips.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestEachChosenStoryIsLoggedWithItsPickAndKey(t *testing.T) {
	logs, err := run(t, stubExecutor{choice: posts.Choice{Chosen: posts.Shortlist{
		{Pick: 1, Title: "BEST", Key: posts.StoryKey{Flag: "dz", Continent: "Africa"}, Countries: []string{"ly", "dz"}},
	}}})

	require.NoError(t, err)
	assert.Contains(t, logs, `msg="chose a story" pick=1 title=BEST flag=dz continent=Africa countries=ly,dz`)
}

func TestEachStorySkippedAsToldIsLogged(t *testing.T) {
	logs, err := run(t, stubExecutor{choice: posts.Choice{Retold: posts.Shortlist{
		{Pick: 1, Title: "ALGERIA VS FRANCE", Key: posts.StoryKey{Flag: "dz", Continent: "Africa"}},
	}}})

	require.NoError(t, err)
	assert.Contains(t, logs, `msg="skipped a story told already" pick=1 title="ALGERIA VS FRANCE" flag=dz continent=Africa`)
}

func TestEachStoryPassedForAFlagToldLessIsLogged(t *testing.T) {
	logs, err := run(t, stubExecutor{choice: posts.Choice{Passed: posts.Shortlist{
		{Pick: 2, Title: "PALAU VS FRANCE", Key: posts.StoryKey{Flag: "pw", Continent: "Europe"}, Countries: []string{"pw", "fr"}},
	}}})

	require.NoError(t, err)
	assert.Contains(t, logs,
		`msg="passed a story for a flag told less lately" pick=2 title="PALAU VS FRANCE" flag=pw continent=Europe countries=pw,fr`)
}

func TestAQuietHalfDayIsLoggedAsSuch(t *testing.T) {
	logs, err := run(t, stubExecutor{})

	require.NoError(t, err)
	assert.Contains(t, logs, `msg="no story is worth a clip"`)
}

func TestAFailureSaysNothing(t *testing.T) {
	logs, err := run(t, stubExecutor{err: errors.New("the file is gone")})

	require.Error(t, err)
	assert.Empty(t, logs)
}
