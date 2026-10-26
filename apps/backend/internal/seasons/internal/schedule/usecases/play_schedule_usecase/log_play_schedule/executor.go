package log_play_schedule

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule/usecases/play_schedule_usecase"
)

func New(inner play_schedule_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_schedule_usecase.Executor
	logger *slog.Logger
}

var _ play_schedule_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (schedule.Turn, bool, error) {
	turn, told, err := l.inner.Execute(ctx)
	if told {
		snapshot := turn.TakeSnapshot()
		attrs := []any{slog.String("set", string(snapshot.Set))}
		if snapshot.Entry != (schedule.EntryID{}) {
			attrs = append(attrs, slog.String("entry", snapshot.Entry.String()))
		}
		l.logger.Warn("the schedule plays", attrs...)
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to play the schedule", slog.Any("error", err))
	}
	return turn, told, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
