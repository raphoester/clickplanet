// Package log_authors logs a caller the chat could not name. The use case refuses the call or serves less.
package log_authors

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
)

func New(inner send_message_usecase.Authors, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  send_message_usecase.Authors
	logger *slog.Logger
}

var _ send_message_usecase.Authors = (*Logged)(nil)

// Author logs a failure at Error: a post or a reaction is refused, a history marks nothing as the caller's.
func (l *Logged) Author(ctx context.Context, account messages.AccountID, ip string) (messages.Author, error) {
	author, err := l.inner.Author(ctx, account, ip)
	if err != nil {
		l.logger.Error("the chat could not ask who is calling",
			slog.String("account", account.String()),
			slog.Any("error", err),
		)
	}
	return author, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
