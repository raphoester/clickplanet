package log_backfill_titles

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/backfill_titles_usecase"
)

func New(inner backfill_titles_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  backfill_titles_usecase.Executor
	logger *slog.Logger
}

var _ backfill_titles_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (backfill_titles_usecase.Backfill, error) {
	backfill, err := l.inner.Execute(ctx)

	switch {
	case err != nil && ctx.Err() == nil:
		l.logger.Error("failed to backfill the titles, retrying at the next boot",
			slog.Any("titles", backfill.Titles), slog.Any("error", err))
	case err == nil && len(backfill.Titles) > 0:
		l.logger.Info("backfilled the titles", slog.Any("titles", backfill.Titles), slog.Int("accounts", backfill.Accounts))
	}

	return backfill, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
