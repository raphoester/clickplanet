package log_converge_rules_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale/usecases/converge_rules_usecase/log_converge_rules"
)

type stubExecutor struct{ err error }

func (s *stubExecutor) Execute(context.Context) error { return s.err }

func TestAFailureIsLoggedOnceUntilItRecovers(t *testing.T) {
	var logs bytes.Buffer
	inner := &stubExecutor{err: errors.New("connection refused")}
	logged := log_converge_rules.New(inner, slog.New(slog.NewTextHandler(&logs, nil)))

	for range 3 {
		require.Error(t, logged.Execute(t.Context()))
	}
	inner.err = nil
	require.NoError(t, logged.Execute(t.Context()))
	require.NoError(t, logged.Execute(t.Context()))

	assert.Equal(t, 1, strings.Count(logs.String(), "level=WARN"))
	assert.Contains(t, logs.String(), `msg="failed to set the season's rules on planet; trying again" error="connection refused"`)
	assert.Equal(t, 1, strings.Count(logs.String(), `level=INFO msg="the season's rules are set on planet again"`))
}

func TestRulesThatHoldLogNothing(t *testing.T) {
	var logs bytes.Buffer

	require.NoError(t, log_converge_rules.New(&stubExecutor{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context()))

	assert.Empty(t, logs.String())
}
