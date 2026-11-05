package log_command

import (
	"context"
	"log/slog"
)

type Executor[C any] interface {
	Execute(ctx context.Context, command C) (bool, error)
}

func New[C any](name string, inner Executor[C], logger *slog.Logger) *Logged[C] {
	return &Logged[C]{name: name, inner: inner, logger: logger}
}

type Logged[C any] struct {
	name   string
	inner  Executor[C]
	logger *slog.Logger
}

func (l *Logged[C]) Execute(ctx context.Context, command C) (bool, error) {
	applied, err := l.inner.Execute(ctx, command)
	if applied {
		l.logger.Info("scoring command applied", slog.String("command", l.name), slog.Any("args", command))
	}
	return applied, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
