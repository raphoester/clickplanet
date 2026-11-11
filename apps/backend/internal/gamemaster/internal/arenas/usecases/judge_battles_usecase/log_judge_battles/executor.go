package log_judge_battles

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/judge_battles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

func New(inner judge_battles_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  judge_battles_usecase.Executor
	logger *slog.Logger
}

var _ judge_battles_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, plan moments.Plan) ([]moments.Judgement, error) {
	judged, err := l.inner.Execute(ctx, plan)
	for _, judgement := range judged {
		l.logger.Warn("a battle is judged", slog.String("round", string(judgement.Round)),
			slog.String("mode", judgement.Mode.String()))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to judge the battle", slog.Any("error", err))
	}
	return judged, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
