package log_record_answer

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
)

type UseCase interface {
	Execute(ctx context.Context, answer ratings.Answer) error
}

func New(inner UseCase, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  UseCase
	logger *slog.Logger
}

func (l *Logged) Execute(ctx context.Context, answer ratings.Answer) error {
	err := l.inner.Execute(ctx, answer)
	if err != nil {
		l.logger.Error("failed to count a quiz answer; the player's answer stands, its question's rating is one short",
			slog.String("question", string(answer.Question)), slog.String("outcome", string(answer.Outcome)),
			slog.Any("error", err))
	}
	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
