package audit_backfill_titles

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/backfill_titles_usecase"
)

func New(inner backfill_titles_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  backfill_titles_usecase.Executor
	logger *slog.Logger
}

var _ backfill_titles_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context) (backfill_titles_usecase.Backfill, error) {
	backfill, err := a.inner.Execute(ctx)
	if err != nil {
		a.logger.Warn("admin title backfill failed", slog.Int("accounts", backfill.Accounts), slog.Any("error", err))
		return backfill, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin title backfill", slog.Int("accounts", backfill.Accounts))
	return backfill, nil
}
