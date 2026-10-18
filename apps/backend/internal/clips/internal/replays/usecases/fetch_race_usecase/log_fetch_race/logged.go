package log_fetch_race

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_race_usecase"
)

func New(inner fetch_race_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  fetch_race_usecase.Executor
	logger *slog.Logger
}

var _ fetch_race_usecase.Executor = (*Logged)(nil)

// A race left out is a warning, not a failure: the run still has its replay, and tells no day of the season.
func (l *Logged) Execute(ctx context.Context) (replays.Race, error) {
	race, err := l.inner.Execute(ctx)
	switch {
	case err != nil:
		l.logger.Warn("left the race out: no day of the season is told", slog.Any("error", err))
	case race.Closed == "":
		l.logger.Info("saved the race: no day closed yet")
	default:
		l.logger.Info("saved the race", slog.String("closed", race.Closed), slog.Time("ended", race.EndedAt))
	}

	return race, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
