package log_play_phase

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/play_phase_usecase"
)

func New(inner play_phase_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_phase_usecase.Executor
	logger *slog.Logger
}

var _ play_phase_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (calendar.Phase, error) {
	played, err := l.inner.Execute(ctx)
	if played != (calendar.Phase{}) {
		l.logger.Info("played a season phase", slog.Uint64("season", uint64(played.Season)),
			slog.String("stage", string(played.Stage)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to play the season phase", slog.Any("error", err))
	}
	return played, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
