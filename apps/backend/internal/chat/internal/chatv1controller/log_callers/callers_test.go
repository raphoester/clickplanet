package log_callers_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/log_callers"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type stubCallers struct {
	account messages.AccountID
	err     error
}

func (s stubCallers) Caller(context.Context, string) (messages.AccountID, error) {
	return s.account, s.err
}

func logged(inner stubCallers) (*log_callers.Logged, *bytes.Buffer) {
	var out bytes.Buffer
	return log_callers.New(inner, slog.New(slog.NewTextHandler(&out, nil))), &out
}

func TestAFailureIsLoggedWithoutTheCookieAndPassedOn(t *testing.T) {
	callers, out := logged(stubCallers{err: errors.New("auth is stuck")})

	_, err := callers.Caller(t.Context(), "cp_sid=secret-token")

	require.EqualError(t, err, "auth is stuck")
	assert.Contains(t, out.String(), "level=ERROR")
	assert.Contains(t, out.String(), "auth is stuck")
	assert.NotContains(t, out.String(), "secret-token", "a cookie signs a browser in")
}

func TestAnAnswerIsPassedOnAndNotLogged(t *testing.T) {
	callers, out := logged(stubCallers{account: messages.AccountID{15: 1}})

	account, err := callers.Caller(t.Context(), "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, messages.AccountID{15: 1}, account)
	assert.Empty(t, out.String())
}
