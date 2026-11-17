package audit_forgive_missed_day

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/forgive_missed_day_usecase"
)

func New(inner forgive_missed_day_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  forgive_missed_day_usecase.Executor
	logger *slog.Logger
}

var _ forgive_missed_day_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context, in forgive_missed_day_usecase.In) (forgive_missed_day_usecase.Forgiven, error) {
	forgiven, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("account_id", in.Account.String()), slog.String("day", in.Day.String()), slog.Bool("dry_run", in.DryRun),
		slog.Group("before", streakAttrs(forgiven.Before)...),
	}
	if err != nil {
		a.logger.Warn("admin missed day forgiveness failed", append(attrs, slog.Any("error", err))...)
		return forgiven, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin missed day forgiven", append(attrs, slog.Group("after", streakAttrs(forgiven.After)...))...)
	return forgiven, nil
}

func streakAttrs(counted stats.Stats) []any {
	return []any{
		slog.Uint64("streak", uint64(counted.Streak().Days())), slog.String("last_day", counted.Streak().LastDay().String()),
		slog.Uint64("best", uint64(counted.StreakBest())),
	}
}
