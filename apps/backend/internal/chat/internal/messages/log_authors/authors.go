// Package log_authors logs a caller the chat could not name. What that costs is the asker's to decide: a post
// is refused, and a history is shown with nobody named on it.
//
// It sits beside rpc_player_authors rather than under one use case, because both the send path and the read
// path go through it.
package log_authors

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// Authors is the player module as the chat asks it: about one account, or about a page of them.
type Authors interface {
	Author(ctx context.Context, account messages.AccountID) (messages.Author, error)
	Authors(ctx context.Context, accounts []messages.AccountID) (map[messages.AccountID]messages.Author, error)
}

func New(inner Authors, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  Authors
	logger *slog.Logger
}

var _ Authors = (*Logged)(nil)

// Author logs a failure at Error: the chat cannot reach the module that names people.
func (l *Logged) Author(ctx context.Context, account messages.AccountID) (messages.Author, error) {
	author, err := l.inner.Author(ctx, account)
	if err != nil {
		l.logger.Error("the chat could not ask who is calling",
			slog.String("account", account.String()),
			slog.Any("error", err),
		)
	}
	return author, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

// Authors logs a failure at Error with how many it was asking about, never with who: a log line is not the
// place for a page of account ids.
func (l *Logged) Authors(
	ctx context.Context,
	accounts []messages.AccountID,
) (map[messages.AccountID]messages.Author, error) {
	found, err := l.inner.Authors(ctx, accounts)
	if err != nil {
		l.logger.Error("the chat could not ask who it is showing",
			slog.Int("accounts", len(accounts)),
			slog.Any("error", err),
		)
	}
	return found, err //nolint:wrapcheck // as above.
}
