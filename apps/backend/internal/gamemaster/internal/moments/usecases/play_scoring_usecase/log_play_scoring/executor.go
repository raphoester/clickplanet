package log_play_scoring

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase"
)

func New(inner play_scoring_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_scoring_usecase.Executor
	logger *slog.Logger
}

var _ play_scoring_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) ([]moments.Command, error) {
	sent, err := l.inner.Execute(ctx)
	for _, command := range sent {
		attrs := []any{
			slog.String("command", command.Action.String()), slog.String("book", string(command.Book)),
			slog.String("at", command.At.UTC().Format(time.RFC3339)),
		}
		if command.Round != "" {
			attrs = append(attrs, slog.String("round", string(command.Round)))
		}
		if command.Times != 0 {
			attrs = append(attrs, slog.Uint64("times", uint64(command.Times)))
		}
		if command.Tiebreak != "" {
			attrs = append(attrs, slog.String("tiebreak", string(command.Tiebreak)))
		}
		l.logger.Warn("scoring is commanded", attrs...)
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to command scoring", slog.Any("error", err))
	}
	return sent, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
