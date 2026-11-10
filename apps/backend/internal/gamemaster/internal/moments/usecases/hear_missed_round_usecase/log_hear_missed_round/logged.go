package log_hear_missed_round

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_missed_round_usecase"
)

func New(inner hear_missed_round_usecase.UseCase, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  hear_missed_round_usecase.UseCase
	logger *slog.Logger
}

var _ hear_missed_round_usecase.UseCase = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, missed moments.MissedRound) (bool, error) {
	kept, err := l.inner.Execute(ctx, missed)
	attrs := []any{slog.String("round", string(missed.Round)), slog.Time("ended_at", missed.EndedAt)}
	switch {
	case err != nil:
		l.logger.Error("failed to keep a round scoring missed", append(attrs, slog.Any("error", err))...)
	case kept:
		l.logger.Warn("scoring missed a round: it never opened, so nothing scores it", attrs...)
	}
	return kept, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
