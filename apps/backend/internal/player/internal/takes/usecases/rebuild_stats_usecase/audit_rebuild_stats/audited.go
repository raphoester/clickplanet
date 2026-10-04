package audit_rebuild_stats

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/usecases/rebuild_stats_usecase"
)

func New(inner rebuild_stats_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  rebuild_stats_usecase.Executor
	logger *slog.Logger
}

var _ rebuild_stats_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context) (rebuild_stats_usecase.Out, error) {
	out, err := a.inner.Execute(ctx)
	if err != nil {
		a.logger.Warn("admin stats rebuild failed", slog.Any("error", err))
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin stats rebuild", slog.Uint64("from", uint64(out.From)))
	return out, nil
}
