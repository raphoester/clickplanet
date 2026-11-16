package log_detonate_mines

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/mines/usecases/detonate_mines_usecase"
)

func New(inner detonate_mines_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  detonate_mines_usecase.Executor
	logger *slog.Logger
}

var _ detonate_mines_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (int, error) {
	detonated, err := l.inner.Execute(ctx)
	if err != nil {
		l.logger.Error("failed to set off the mines a take tripped: the one that failed is lost, the others wait for the next tick",
			slog.Int("detonated", detonated), slog.Any("error", err))
	}
	return detonated, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
