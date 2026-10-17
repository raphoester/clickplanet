package log_refusals

import (
	"context"
	"log/slog"
)

type Operators interface {
	Operator(ctx context.Context, cookieHeader string) (string, error)
}

func New(inner Operators, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  Operators
	logger *slog.Logger
}

func (l *Logged) Operator(ctx context.Context, cookieHeader string) (string, error) {
	account, err := l.inner.Operator(ctx, cookieHeader)
	if err != nil {
		l.logger.Warn("operator refused", slog.Any("error", err))
	}

	return account, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
