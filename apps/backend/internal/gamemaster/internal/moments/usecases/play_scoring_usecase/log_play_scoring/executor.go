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

func (l *Logged) Execute(ctx context.Context, plan moments.Plan) ([]play_scoring_usecase.Call, error) {
	calls, err := l.inner.Execute(ctx, plan)
	for _, call := range calls {
		attrs := attrsOf(call.Command)
		switch {
		case call.Refusal != nil:
			l.logger.Error("scoring refused a command", append(attrs, slog.Any("error", call.Refusal))...)
		case call.Command.Action == moments.CloseRound || call.Command.Action == moments.CloseBook:
			l.logger.Warn("scoring answered", append(attrs, slog.Bool("missed", call.Answer.Missed),
				slog.String("ended_at", call.Answer.EndedAt.UTC().Format(time.RFC3339)))...)
		default:
			l.logger.Warn("scoring answered", attrs...)
		}
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to call scoring", slog.Any("error", err))
	}
	return calls, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

func attrsOf(command moments.Command) []any {
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
	if command.Ranking != "" {
		attrs = append(attrs, slog.String("ranking", string(command.Ranking)))
	}
	if command.Tiebreak != "" {
		attrs = append(attrs, slog.String("tiebreak", string(command.Tiebreak)))
	}
	return attrs
}
