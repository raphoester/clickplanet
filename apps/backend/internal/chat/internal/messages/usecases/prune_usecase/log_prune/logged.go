// Package log_prune logs what each chat prune deleted, and why one failed.
package log_prune

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/prune_usecase"
)

func New(inner prune_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  prune_usecase.Executor
	logger *slog.Logger
}

var _ prune_usecase.Executor = (*Logged)(nil)

// Execute logs a failure at Error, unless the process is stopping, and a prune that deleted something at Info.
func (l *Logged) Execute(ctx context.Context) (int64, error) {
	deleted, err := l.inner.Execute(ctx)

	switch {
	case err != nil && ctx.Err() == nil:
		l.logger.Error("failed to prune the chat, retrying next tick", slog.Int64("deleted", deleted), slog.Any("error", err))
	case deleted > 0:
		l.logger.Info("pruned the chat", slog.Int64("deleted", deleted))
	}

	return deleted, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
