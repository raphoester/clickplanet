package log_refund_rejected

import (
	"context"
	"log/slog"
)

type Executor interface {
	Execute(ctx context.Context, submission string) error
}

func New(inner Executor, logger *slog.Logger) Log {
	return Log{inner: inner, logger: logger}
}

// Log says each refund of a rejected flag: money goes back, and the operator answers for it.
type Log struct {
	inner  Executor
	logger *slog.Logger
}

func (l Log) Execute(ctx context.Context, submission string) error {
	err := l.inner.Execute(ctx, submission)
	if err != nil {
		l.logger.Error("failed to refund a rejected flag", slog.String("submission", submission), slog.Any("error", err))
		return err //nolint:wrapcheck // a decorator adds nothing.
	}
	l.logger.Info("a rejected flag refunded", slog.String("submission", submission))
	return nil
}
