package log_usernames_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/log_usernames"
)

type stubUsernames struct {
	username string
	found    bool
	err      error
}

func (s stubUsernames) Username(context.Context, messages.AccountID) (string, bool, error) {
	return s.username, s.found, s.err
}

func logged(inner stubUsernames) (*log_usernames.Logged, *bytes.Buffer) {
	var out bytes.Buffer
	return log_usernames.New(inner, slog.New(slog.NewTextHandler(&out, nil))), &out
}

func TestAFailureIsLoggedAndPassedOn(t *testing.T) {
	usernames, out := logged(stubUsernames{err: errors.New("player is off")})

	_, _, err := usernames.Username(t.Context(), messages.AccountID{15: 1})

	require.EqualError(t, err, "player is off")
	assert.Contains(t, out.String(), "level=WARN")
	assert.Contains(t, out.String(), "player is off")
	assert.Contains(t, out.String(), messages.AccountID{15: 1}.String())
}

func TestAnAnswerIsPassedOnAndNotLogged(t *testing.T) {
	usernames, out := logged(stubUsernames{username: "Ada", found: true})

	username, found, err := usernames.Username(t.Context(), messages.AccountID{15: 1})

	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Ada", username)
	assert.Empty(t, out.String())
}
