package log_authors_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/log_authors"
)

type stubAuthors struct {
	author messages.Author
	err    error
}

func (s stubAuthors) Author(context.Context, messages.AccountID, string) (messages.Author, error) {
	return s.author, s.err
}

func logged(inner stubAuthors) (*log_authors.Logged, *bytes.Buffer) {
	var out bytes.Buffer
	return log_authors.New(inner, slog.New(slog.NewTextHandler(&out, nil))), &out
}

func TestAFailureIsLoggedAndPassedOn(t *testing.T) {
	authors, out := logged(stubAuthors{err: errors.New("player is stuck")})

	_, err := authors.Author(t.Context(), messages.AccountID{15: 1}, "1.2.3.4")

	require.EqualError(t, err, "player is stuck")
	assert.Contains(t, out.String(), "level=ERROR")
	assert.Contains(t, out.String(), "player is stuck")
	assert.Contains(t, out.String(), messages.AccountID{15: 1}.String())
	assert.NotContains(t, out.String(), "1.2.3.4", "the address is not logged")
}

func TestAnAnswerIsPassedOnAndNotLogged(t *testing.T) {
	authors, out := logged(stubAuthors{author: messages.Author{Username: "Ada", Tag: "a1b2c3"}})

	author, err := authors.Author(t.Context(), messages.AccountID{15: 1}, "1.2.3.4")

	require.NoError(t, err)
	assert.Equal(t, messages.Author{Username: "Ada", Tag: "a1b2c3"}, author)
	assert.Empty(t, out.String())
}
