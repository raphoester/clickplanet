package audit_backfill_addresses

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/backfill_addresses_usecase"
)

func New(inner backfill_addresses_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  backfill_addresses_usecase.Executor
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, dryRun bool) (notes.BackfillCounts, error) {
	counts, err := a.inner.Execute(ctx, dryRun)
	if err != nil {
		a.logger.Warn("admin reply address backfill failed", slog.Bool("dry_run", dryRun), slog.Any("error", err))
		return counts, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin reply address backfill", slog.Bool("dry_run", dryRun),
		slog.Int("filled", counts.Filled), slog.Int("guests", counts.Guests), slog.Int("no_address", counts.NoAddress))
	return counts, nil
}
