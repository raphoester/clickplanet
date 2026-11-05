package log_purge_deletions

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/purge_deletions_usecase"
)

func New(inner purge_deletions_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  purge_deletions_usecase.Executor
	logger *slog.Logger
}

var _ purge_deletions_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (int, error) {
	purged, err := l.inner.Execute(ctx)

	switch {
	case err != nil && ctx.Err() == nil:
		l.logger.Error("failed to purge the accounts whose deletion was requested", slog.Int("purged", purged), slog.Any("error", err))
	case purged > 0:
		l.logger.Info("purged accounts whose deletion was requested", slog.Int("purged", purged))
	}

	return purged, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
