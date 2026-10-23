package log_fetch_season

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_season_usecase"
)

func New(inner fetch_season_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  fetch_season_usecase.Executor
	logger *slog.Logger
}

var _ fetch_season_usecase.Executor = (*Logged)(nil)

// A season left out is a warning, not a failure: the run still has its replay, and calls no country up.
func (l *Logged) Execute(ctx context.Context) (replays.Season, error) {
	season, err := l.inner.Execute(ctx)
	switch {
	case err != nil:
		l.logger.Warn("left the season out: no country is called up", slog.Any("error", err))
	case season.FinaleStartsAt.IsZero():
		l.logger.Info("saved the season: none is on")
	default:
		l.logger.Info("saved the season", slog.Time("finale", season.FinaleStartsAt))
	}

	return season, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
