package log_callers

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Callers interface {
	Caller(ctx context.Context, cookie string) (messages.AccountID, error)
}

func New(inner Callers, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  Callers
	logger *slog.Logger
}

var _ Callers = (*Logged)(nil)

func (l *Logged) Caller(ctx context.Context, cookie string) (messages.AccountID, error) {
	account, err := l.inner.Caller(ctx, cookie)
	if err != nil {
		l.logger.Error("the chat could not ask whose cookie it was sent", slog.Any("error", err))
	}
	return account, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
