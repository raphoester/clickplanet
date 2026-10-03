package audit_reconcile_titles

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/reconcile_titles_usecase"
)

func New(inner reconcile_titles_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  reconcile_titles_usecase.Executor
	logger *slog.Logger
}

var _ reconcile_titles_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context) (reconcile_titles_usecase.Reconciled, error) {
	reconciled, err := a.inner.Execute(ctx)

	attrs := []any{slog.Int("granted", reconciled.Granted), slog.Int("revoked", reconciled.Revoked)}
	if err != nil {
		a.logger.Warn("admin title reconciliation failed", append(attrs, slog.Any("error", err))...)
		return reconciled, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin title reconciliation", attrs...)
	return reconciled, nil
}
