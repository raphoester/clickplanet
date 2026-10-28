package log_fetch_turn

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_turn_usecase"
)

func New(inner fetch_turn_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  fetch_turn_usecase.Executor
	logger *slog.Logger
}

var _ fetch_turn_usecase.Executor = (*Logged)(nil)

// A turn left out is a warning, not a failure: the run still has its replay, and tells no round of the season.
func (l *Logged) Execute(ctx context.Context) (replays.Turn, error) {
	turn, err := l.inner.Execute(ctx)
	switch {
	case err != nil:
		l.logger.Warn("left the turn out: no round of the season is told, no country called up", slog.Any("error", err))
	case turn.Debriefed == "":
		l.logger.Info("saved the turn: nothing scored yet", slog.Time("finale", turn.FinaleAt))
	default:
		l.logger.Info("saved the turn", slog.String("debriefed", turn.Debriefed), slog.Time("ended", turn.EndedAt),
			slog.Time("finale", turn.FinaleAt))
	}

	return turn, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
