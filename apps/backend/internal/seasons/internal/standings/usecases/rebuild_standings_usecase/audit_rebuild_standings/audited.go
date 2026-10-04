package audit_rebuild_standings

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/rebuild_standings_usecase"
)

func New(inner rebuild_standings_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  rebuild_standings_usecase.Executor
	logger *slog.Logger
}

var _ rebuild_standings_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context) (rebuild_standings_usecase.Out, error) {
	out, err := a.inner.Execute(ctx)
	if err != nil {
		a.logger.Warn("admin standings rebuild failed", slog.Any("error", err))
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin standings rebuild", slog.Uint64("from", uint64(out.From)))
	return out, nil
}
