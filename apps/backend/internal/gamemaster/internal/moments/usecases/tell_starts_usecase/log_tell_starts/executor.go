package log_tell_starts

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/tell_starts_usecase"
)

func New(inner tell_starts_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  tell_starts_usecase.Executor
	logger *slog.Logger
}

var _ tell_starts_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) ([]moments.Moment, error) {
	told, err := l.inner.Execute(ctx)
	for _, moment := range told {
		switch started := moment.(type) {
		case moments.SeasonStarted:
			l.logger.Info("season told", slog.String("book", string(moments.BookOf(started.Window.Number))),
				slog.String("starts_at", stamp(started.Window.StartsAt)), slog.String("ends_at", stamp(started.Window.EndsAt)))
		case moments.DayStarted:
			l.logger.Info("day told", slog.String("round", string(moments.DayRoundOf(started.Day))),
				slog.String("starts_at", stamp(started.Day.StartsAt)), slog.String("ends_at", stamp(started.Day.EndsAt)))
		case moments.BattleStarted:
			l.logger.Info("battle told", slog.String("round", string(started.Battle.Round())),
				slog.String("starts_at", stamp(started.Battle.StartsAt)), slog.String("ends_at", stamp(started.Battle.EndsAt)))
		}
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to tell what starts", slog.Any("error", err))
	}
	return told, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

func stamp(at time.Time) string { return at.UTC().Format(time.RFC3339) }
