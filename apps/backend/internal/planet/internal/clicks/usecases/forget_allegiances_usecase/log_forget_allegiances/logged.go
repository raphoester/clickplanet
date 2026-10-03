// Package log_forget_allegiances logs what each forget of the faded allegiances deleted, and why one failed.
package log_forget_allegiances

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/forget_allegiances_usecase"
)

func New(inner forget_allegiances_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  forget_allegiances_usecase.Executor
	logger *slog.Logger
}

var _ forget_allegiances_usecase.Executor = (*Logged)(nil)

// Execute logs a failure at Error, unless the process is stopping, and a forget that deleted something at Info.
func (l *Logged) Execute(ctx context.Context) (int64, error) {
	deleted, err := l.inner.Execute(ctx)

	switch {
	case err != nil && ctx.Err() == nil:
		l.logger.Error("failed to forget the faded allegiances, retrying next tick", slog.Any("error", err))
	case deleted > 0:
		l.logger.Info("forgot the faded allegiances", slog.Int64("deleted", deleted))
	}

	return deleted, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
