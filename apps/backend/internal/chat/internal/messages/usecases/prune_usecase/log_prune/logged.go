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
