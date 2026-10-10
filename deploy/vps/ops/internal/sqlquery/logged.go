package sqlquery

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-ops/internal/access"
)

func NewLogged(inner Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  Executor
	logger *slog.Logger
}

func (l *Logged) Execute(ctx context.Context, query Query) (Result, error) {
	start := time.Now()
	result, err := l.inner.Execute(ctx, query)

	attributes := []any{
		slog.String("caller", string(access.CallerFrom(ctx))),
		slog.String("statement", query.Statement),
		slog.Duration("took", time.Since(start)),
	}
	if err != nil {
		l.logger.Warn("a statement failed", append(attributes, slog.Any("error", err))...)
		return result, err
	}
	l.logger.Info("ran a statement", append(
		attributes,
		slog.Int("rows", len(result.Rows)),
		slog.Bool("truncated", result.Truncated),
	)...)
	return result, nil
}
