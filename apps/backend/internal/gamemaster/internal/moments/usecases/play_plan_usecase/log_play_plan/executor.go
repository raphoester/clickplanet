package log_play_plan

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_plan_usecase"
)

func New(inner play_plan_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_plan_usecase.Executor
	logger *slog.Logger
}

var _ play_plan_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) error {
	err := l.inner.Execute(ctx)
	// Each step logs its own failure.
	if errors.Is(err, play_plan_usecase.ErrUnplanned) && ctx.Err() == nil {
		l.logger.Error("failed to plan the books", slog.Any("error", err))
	}
	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
