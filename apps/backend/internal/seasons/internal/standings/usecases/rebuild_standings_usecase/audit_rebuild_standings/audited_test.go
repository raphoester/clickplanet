package audit_rebuild_standings_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/rebuild_standings_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/rebuild_standings_usecase/audit_rebuild_standings"
)

type stubExecutor struct {
	out rebuild_standings_usecase.Out
	err error
}

func (s stubExecutor) Execute(context.Context) (rebuild_standings_usecase.Out, error) {
	return s.out, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_rebuild_standings.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestEveryRebuildIsLoggedWithWhereItReplaysFrom(t *testing.T) {
	logs, err := run(t, stubExecutor{out: rebuild_standings_usecase.Out{From: 42}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin standings rebuild" from=42`)
}

func TestAFailedRebuildIsLogged(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubExecutor{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin standings rebuild failed" error="postgres is down"`)
}
