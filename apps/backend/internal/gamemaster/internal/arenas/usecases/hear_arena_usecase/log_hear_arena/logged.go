package log_hear_arena

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/hear_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

func New(inner hear_arena_usecase.UseCase, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  hear_arena_usecase.UseCase
	logger *slog.Logger
}

var _ hear_arena_usecase.UseCase = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, arena moments.RoundID) (bool, error) {
	kept, err := l.inner.Execute(ctx, arena)
	attrs := []any{slog.String("arena", string(arena))}
	switch {
	case err != nil:
		l.logger.Error("failed to keep an arena closed", append(attrs, slog.Any("error", err))...)
	case kept:
		l.logger.Warn("an arena is closed: the map from before it is back", attrs...)
	default:
		l.logger.Info("the arena was kept closed already", attrs...)
	}
	return kept, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
