package log_prune_grants

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants/usecases/prune_grants_usecase"
)

func New(inner prune_grants_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  prune_grants_usecase.Executor
	logger *slog.Logger
}

var _ prune_grants_usecase.Executor = (*Logged)(nil)

// A prune that deleted grants logs nothing: every grant ends there, once its join is done.
func (l *Logged) Execute(ctx context.Context) (int, error) {
	pruned, err := l.inner.Execute(ctx)

	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to prune the expired Discord grants", slog.Int("pruned", pruned), slog.Any("error", err))
	}

	return pruned, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
