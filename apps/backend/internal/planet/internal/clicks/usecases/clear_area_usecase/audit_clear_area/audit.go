package audit_clear_area

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/clear_area_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in clear_area_usecase.In) (clear_area_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in clear_area_usecase.In) (clear_area_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.Uint64("tile", uint64(in.Tile)), slog.Uint64("rings", uint64(in.Rings)), slog.Bool("dryRun", in.DryRun),
		slog.Int("tiles", out.Tiles), slog.Int("owned", out.Owned), slog.Int("cleared", out.Cleared),
	}

	if err != nil {
		a.logger.Warn("admin area clear failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin area clear", attrs...)

	return out, nil
}
