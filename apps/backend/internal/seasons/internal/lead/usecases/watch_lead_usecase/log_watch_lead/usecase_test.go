package log_watch_lead_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead/usecases/watch_lead_usecase/log_watch_lead"
)

type stubExecutor struct{ err error }

func (s *stubExecutor) Execute(context.Context) error { return s.err }

func TestAFailureIsLoggedOnceUntilItRecovers(t *testing.T) {
	var logs bytes.Buffer
	inner := &stubExecutor{err: errors.New("connection refused")}
	logged := log_watch_lead.New(inner, slog.New(slog.NewTextHandler(&logs, nil)))

	for range 3 {
		require.Error(t, logged.Execute(t.Context()))
	}
	inner.err = nil
	require.NoError(t, logged.Execute(t.Context()))

	assert.Equal(t, 1, strings.Count(logs.String(), "level=WARN"))
	assert.Contains(t, logs.String(), `msg="failed to follow the lead; trying again" error="connection refused"`)
	assert.Contains(t, logs.String(), `level=INFO msg="following the lead again"`)
}

func TestAWatchThatWorksLogsNothing(t *testing.T) {
	var logs bytes.Buffer

	require.NoError(t, log_watch_lead.New(&stubExecutor{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context()))

	assert.Empty(t, logs.String())
}
