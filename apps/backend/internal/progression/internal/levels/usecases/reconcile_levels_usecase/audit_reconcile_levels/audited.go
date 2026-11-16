package audit_reconcile_levels

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/usecases/reconcile_levels_usecase"
)

func New(inner reconcile_levels_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  reconcile_levels_usecase.Executor
	logger *slog.Logger
}

var _ reconcile_levels_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context, dryRun bool) (reconcile_levels_usecase.Reconciled, error) {
	reconciled, err := a.inner.Execute(ctx, dryRun)

	attrs := []any{
		slog.Bool("dry_run", dryRun), slog.Int("raised", reconciled.Raised),
		slog.Int("removed", reconciled.Removed),
	}
	if err != nil {
		a.logger.Warn("admin level reconciliation failed", append(attrs, slog.Any("error", err))...)
		return reconciled, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin level reconciliation", attrs...)
	return reconciled, nil
}
