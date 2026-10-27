package log_close_ended_rounds

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/close_ended_rounds_usecase"
)

const lateAfter = time.Second

func New(inner close_ended_rounds_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  close_ended_rounds_usecase.Executor
	logger *slog.Logger
}

var _ close_ended_rounds_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (close_ended_rounds_usecase.Report, error) {
	report, err := l.inner.Execute(ctx)

	for _, round := range report.Closed {
		level, message := slog.LevelInfo, "closed a round"
		late := report.ReadAt.Sub(round.Round.EndsAt)
		if late >= lateAfter {
			level, message = slog.LevelWarn, "closed a round late, from the map as read then"
		}
		l.logger.Log(ctx, level, message,
			slog.Uint64("season", uint64(round.Round.Season)),
			slog.Uint64("number", uint64(round.Number)),
			slog.Time("endsAt", round.Round.EndsAt),
			slog.Bool("finale", round.Round.Finale),
			slog.Int("ranked", len(round.Results)),
			slog.Duration("late", late))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to close the rounds that ended", slog.Any("error", err))
	}

	return report, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
