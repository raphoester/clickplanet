package log_fetch_round

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_round_usecase"
)

func New(inner fetch_round_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  fetch_round_usecase.Executor
	logger *slog.Logger
}

var _ fetch_round_usecase.Executor = (*Logged)(nil)

// A round left out is a warning, not a failure: the run still has its replay, and tells no round of the season.
func (l *Logged) Execute(ctx context.Context, turn replays.Turn) (replays.Round, error) {
	round, err := l.inner.Execute(ctx, turn)
	switch {
	case err != nil:
		l.logger.Warn("left the round out: no round of the season is told", slog.Any("error", err))
	case round.ID == "":
		l.logger.Info("saved the round: none was debriefed")
	default:
		l.logger.Info("saved the round", slog.String("round", round.ID.String()))
	}

	return round, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
