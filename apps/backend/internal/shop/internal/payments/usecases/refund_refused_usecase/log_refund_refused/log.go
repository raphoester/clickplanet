package log_refund_refused

import (
	"context"
	"log/slog"
)

type Executor interface {
	Execute(ctx context.Context, grant string) error
}

func New(inner Executor, logger *slog.Logger) Log {
	return Log{inner: inner, logger: logger}
}

// Log says each refund of a payment the flags module refused.
type Log struct {
	inner  Executor
	logger *slog.Logger
}

func (l Log) Execute(ctx context.Context, grant string) error {
	err := l.inner.Execute(ctx, grant)
	if err != nil {
		l.logger.Error("failed to refund a payment refused", slog.String("grant", grant), slog.Any("error", err))
		return err //nolint:wrapcheck // a decorator adds nothing.
	}
	l.logger.Info("a payment refused refunded", slog.String("grant", grant))
	return nil
}
