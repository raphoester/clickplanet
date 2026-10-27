package log_close_ended_rounds

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_ended_rounds_usecase"
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
		level, late := slog.LevelInfo, report.ReadAt.Sub(round.Round.EndsAt)
		if late >= lateAfter {
			level = slog.LevelWarn
		}
		l.logger.Log(ctx, level, "closed a round",
			slog.Uint64("season", uint64(round.Round.Season)),
			slog.Uint64("number", uint64(round.Number)),
			slog.Time("endsAt", round.Round.EndsAt),
			slog.Bool("finale", round.Round.Finale),
			slog.Bool("own", round.Round.Own),
			slog.Uint64("times", uint64(round.Round.Times)),
			slog.Int("ranked", len(round.Results)),
			slog.Duration("late", late))
	}
	for _, season := range report.Ended {
		l.logger.Warn("ended a season",
			slog.Uint64("season", uint64(season.Season)),
			slog.Time("endedAt", season.EndedAt),
			slog.Int("ranked", len(season.Table)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to close the rounds that ended", slog.Any("error", err))
	}

	return report, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
