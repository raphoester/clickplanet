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
	chosen posts.Shortlist
	err    error
}

func (s stubExecutor) Execute(context.Context) (posts.Shortlist, error) {
	return s.chosen, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_choose_clips.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestEachChosenStoryIsLoggedWithItsPick(t *testing.T) {
	logs, err := run(t, stubExecutor{chosen: posts.ShortlistOf([]string{"BEST", "SECOND"})})

	require.NoError(t, err)
	assert.Contains(t, logs, `msg="chose a story" pick=1 title=BEST`)
	assert.Contains(t, logs, `msg="chose a story" pick=2 title=SECOND`)
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
