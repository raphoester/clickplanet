// Package audit_paint_random logs every random paint, dry runs and failures included.
package audit_paint_random

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in paint_random_tiles_usecase.In) (paint_random_tiles_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

// Execute logs at Warn: the log is the only record that these tiles did not change hands through play.
func (a *Audited) Execute(ctx context.Context, in paint_random_tiles_usecase.In) (paint_random_tiles_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("flag", in.Flag), slog.String("area", in.Area), slog.Int("count", in.Count),
		slog.Float64("proximity", in.Proximity), slog.Bool("dryRun", in.DryRun),
		slog.Int("eligible", out.Eligible), slog.Int("picked", out.Picked), slog.Int("painted", out.Painted),
	}

	if err != nil {
		a.logger.Warn("admin random paint failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin random paint", attrs...)

	return out, nil
}
