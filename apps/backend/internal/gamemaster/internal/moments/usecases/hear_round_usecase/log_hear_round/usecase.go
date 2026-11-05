package log_hear_round

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_round_usecase"
)

func New(inner hear_round_usecase.UseCase, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  hear_round_usecase.UseCase
	logger *slog.Logger
}

var _ hear_round_usecase.UseCase = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, scored moments.ScoredRound) (hear_round_usecase.Heard, error) {
	heard, err := l.inner.Execute(ctx, scored)
	attrs := []any{slog.String("round", string(scored.Round)), slog.String("book", string(scored.Book))}
	switch {
	case err != nil:
		l.logger.Error("failed to hear a round scoring scored", append(attrs, slog.Any("error", err))...)
	case heard == hear_round_usecase.Told:
		l.logger.Info("scoring scored a round", attrs...)
	case heard == hear_round_usecase.Unknown:
		l.logger.Warn("scoring scored a round no day or battle has: dropped", attrs...)
	}
	return heard, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
