package log_prune

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/prune_usecase"
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
	anonymized, err := l.inner.Execute(ctx)

	switch {
	case err != nil && ctx.Err() == nil:
		l.logger.Error("failed to prune the support module, retrying next tick",
			slog.Int64("anonymized", anonymized), slog.Any("error", err))
	case anonymized > 0:
		l.logger.Info("pruned the support module", slog.Int64("anonymized", anonymized))
	}

	return anonymized, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
