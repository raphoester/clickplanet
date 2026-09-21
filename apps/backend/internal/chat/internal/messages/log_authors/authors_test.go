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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/log_authors"
)

type stubAuthors struct {
	author messages.Author
	err    error
}

func (s stubAuthors) Author(context.Context, messages.AccountID) (messages.Author, error) {
	return s.author, s.err
}

func (s stubAuthors) Authors(
	context.Context,
	[]messages.AccountID,
) (map[messages.AccountID]messages.Author, error) {
	return map[messages.AccountID]messages.Author{{15: 1}: s.author}, s.err
}

func logged(inner stubAuthors) (*log_authors.Logged, *bytes.Buffer) {
	var out bytes.Buffer
	return log_authors.New(inner, slog.New(slog.NewTextHandler(&out, nil))), &out
}

func TestAFailureIsLoggedAndPassedOn(t *testing.T) {
	authors, out := logged(stubAuthors{err: errors.New("player is stuck")})

	_, err := authors.Author(t.Context(), messages.AccountID{15: 1})

	require.EqualError(t, err, "player is stuck")
	assert.Contains(t, out.String(), "level=ERROR")
	assert.Contains(t, out.String(), "player is stuck")
	assert.Contains(t, out.String(), messages.AccountID{15: 1}.String())
}

func TestAnAnswerIsPassedOnAndNotLogged(t *testing.T) {
	authors, out := logged(stubAuthors{author: messages.Author{Name: "Ada"}})

	author, err := authors.Author(t.Context(), messages.AccountID{15: 1})

	require.NoError(t, err)
	assert.Equal(t, messages.Author{Name: "Ada"}, author)
	assert.Empty(t, out.String())
}

func TestAFailedPageIsLoggedWithHowManyAndNotWithWho(t *testing.T) {
	authors, out := logged(stubAuthors{err: errors.New("player is stuck")})

	_, err := authors.Authors(t.Context(), []messages.AccountID{{15: 1}, {15: 2}})

	require.EqualError(t, err, "player is stuck")
	assert.Contains(t, out.String(), "level=ERROR")
	assert.Contains(t, out.String(), "accounts=2")
	assert.NotContains(t, out.String(), messages.AccountID{15: 1}.String(), "a log line is not a page of ids")
}

func TestAnAnsweredPageIsPassedOnAndNotLogged(t *testing.T) {
	authors, out := logged(stubAuthors{author: messages.Author{Name: "Ada"}})

	found, err := authors.Authors(t.Context(), []messages.AccountID{{15: 1}})

	require.NoError(t, err)
	assert.Equal(t, map[messages.AccountID]messages.Author{{15: 1}: {Name: "Ada"}}, found)
	assert.Empty(t, out.String())
}
