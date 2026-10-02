package log_authors

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

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
