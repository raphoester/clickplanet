// Package log_prune_guests logs what each prune deleted, and why one failed.
package log_prune_guests

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/prune_guests_usecase"
)

func New(inner prune_guests_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  prune_guests_usecase.Executor
	logger *slog.Logger
}

var _ prune_guests_usecase.Executor = (*Logged)(nil)

// Execute logs a failure at Error, unless the process is stopping, and a prune that deleted something at Info.
func (l *Logged) Execute(ctx context.Context) (int, error) {
	pruned, err := l.inner.Execute(ctx)

	switch {
	case err != nil && ctx.Err() == nil:
		l.logger.Error("failed to prune the idle guests", slog.Int("pruned", pruned), slog.Any("error", err))
	case pruned > 0:
		l.logger.Info("pruned idle guests", slog.Int("pruned", pruned))
	}

	return pruned, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
