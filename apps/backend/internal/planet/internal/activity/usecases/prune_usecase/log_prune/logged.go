package log_prune

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/usecases/prune_usecase"
)

func New(inner prune_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  prune_usecase.Executor
	logger *slog.Logger
}

var _ prune_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (prune_usecase.Pruned, error) {
	pruned, err := l.inner.Execute(ctx)

	expired, excess := slog.Int64("expired", pruned.Expired), slog.Int64("excess", pruned.Excess)

	switch {
	case err != nil && ctx.Err() == nil:
		l.logger.Error("failed to prune the activity, retrying next tick", expired, excess, slog.Any("error", err))
	case pruned.Excess > 0:
		l.logger.Warn("the activity is full, the oldest events went before the retention", expired, excess)
	case pruned.Expired > 0:
		l.logger.Info("pruned the activity", expired, excess)
	}

	return pruned, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
