package audit_recount_stats

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/recount_stats_usecase"
)

func New(inner recount_stats_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  recount_stats_usecase.Executor
	logger *slog.Logger
}

var _ recount_stats_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context) (int, error) {
	recounted, err := a.inner.Execute(ctx)
	if err != nil {
		a.logger.Warn("admin stats recount failed", slog.Int("recounted", recounted), slog.Any("error", err))
		return recounted, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin stats recount", slog.Int("recounted", recounted))
	return recounted, nil
}
