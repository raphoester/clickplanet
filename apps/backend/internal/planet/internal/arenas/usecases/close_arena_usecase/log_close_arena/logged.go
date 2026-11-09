package log_close_arena

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/close_arena_usecase"
)

func New(inner close_arena_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  close_arena_usecase.Executor
	logger *slog.Logger
}

var _ close_arena_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, arena arenas.ID) (close_arena_usecase.Closed, error) {
	closed, err := l.inner.Execute(ctx, arena)
	attrs := []any{slog.String("arena", string(arena))}
	switch {
	case err != nil:
		l.logger.Error("failed to close an arena", append(attrs, slog.Any("error", err))...)
	case closed == close_arena_usecase.Restored:
		l.logger.Warn("an arena closed: the world's map plays again", attrs...)
	case closed == close_arena_usecase.Unopened:
		l.logger.Warn("an arena that never opened closed", attrs...)
	case closed == close_arena_usecase.Already:
		l.logger.Info("the arena was closed already", attrs...)
	}
	return closed, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
