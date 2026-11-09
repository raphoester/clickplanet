package log_open_arena

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/open_arena_usecase"
)

func New(inner open_arena_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  open_arena_usecase.Executor
	logger *slog.Logger
}

var _ open_arena_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, command open_arena_usecase.Command) (bool, error) {
	opened, err := l.inner.Execute(ctx, command)
	attrs := []any{slog.String("arena", string(command.Arena)), slog.Time("endsAt", command.EndsAt)}
	switch {
	case errors.Is(err, arenas.ErrRefused):
		l.logger.Error("refused to open an arena", append(attrs, slog.Any("error", err))...)
	case err != nil:
		l.logger.Error("failed to open an arena", append(attrs, slog.Any("error", err))...)
	case opened:
		l.logger.Warn("an arena is open: the map is blank", attrs...)
	default:
		l.logger.Info("the arena was open already", attrs...)
	}
	return opened, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
