package audit_recount_stats_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/recount_stats_usecase/audit_recount_stats"
)

type stubExecutor struct {
	recounted int
	err       error
}

func (s stubExecutor) Execute(context.Context) (int, error) { return s.recounted, s.err }

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_recount_stats.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestEveryRecountIsLoggedWithHowManyAccountsItChanged(t *testing.T) {
	logs, err := run(t, stubExecutor{recounted: 164})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin stats recount" recounted=164`)
}

func TestAFailureIsLoggedWithWhatWasDoneBeforeIt(t *testing.T) {
	cause := errors.New("planet is down")

	logs, err := run(t, stubExecutor{recounted: 500, err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin stats recount failed" recounted=500`)
}
