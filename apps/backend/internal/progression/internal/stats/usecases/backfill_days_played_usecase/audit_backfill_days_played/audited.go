package audit_backfill_days_played

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/backfill_days_played_usecase"
)

func New(inner backfill_days_played_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  backfill_days_played_usecase.Executor
	logger *slog.Logger
}

var _ backfill_days_played_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context, dryRun bool) (backfill_days_played_usecase.Backfilled, error) {
	backfilled, err := a.inner.Execute(ctx, dryRun)

	attrs := []any{slog.Bool("dry_run", dryRun), slog.Int("accounts", backfilled.Accounts), slog.Int("changed", backfilled.Changed)}
	if err != nil {
		a.logger.Warn("admin days played backfill failed", append(attrs, slog.Any("error", err))...)
		return backfilled, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin days played backfill", attrs...)
	return backfilled, nil
}
