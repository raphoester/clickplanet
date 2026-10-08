package log_take_snapshot

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/take_snapshot_usecase"
)

func New(inner take_snapshot_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  take_snapshot_usecase.Executor
	logger *slog.Logger
}

var _ take_snapshot_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) ([]rounds.Round, error) {
	closed, err := l.inner.Execute(ctx)

	for _, round := range closed {
		l.logger.Info("closed a round",
			slog.Uint64("season", uint64(round.Season)),
			slog.Time("endsAt", round.EndsAt),
			slog.Bool("finale", round.Finale))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to take the snapshot", slog.Any("error", err))
	}

	return closed, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
