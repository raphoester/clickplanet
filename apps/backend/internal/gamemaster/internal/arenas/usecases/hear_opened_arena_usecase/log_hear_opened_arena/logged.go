package log_hear_opened_arena

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/hear_opened_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

func New(inner hear_opened_arena_usecase.UseCase, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  hear_opened_arena_usecase.UseCase
	logger *slog.Logger
}

var _ hear_opened_arena_usecase.UseCase = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, arena moments.RoundID) (bool, error) {
	kept, err := l.inner.Execute(ctx, arena)
	attrs := []any{slog.String("arena", string(arena))}
	switch {
	case err != nil:
		l.logger.Error("failed to keep an arena opened", append(attrs, slog.Any("error", err))...)
	case kept:
		l.logger.Warn("an arena is opened: its battle's round opens on the blank map", attrs...)
	default:
		l.logger.Info("the arena was kept opened already", attrs...)
	}
	return kept, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
