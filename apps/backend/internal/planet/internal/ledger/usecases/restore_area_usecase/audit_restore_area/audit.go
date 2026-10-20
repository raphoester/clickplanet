package audit_restore_area

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/restore_area_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in restore_area_usecase.In) (restore_area_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in restore_area_usecase.In) (restore_area_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.Uint64("tile", uint64(in.Tile)), slog.Uint64("rings", uint64(in.Rings)),
		slog.String("before", in.Before.UTC().Format(time.RFC3339)), slog.Bool("dryRun", in.DryRun),
		slog.Int("tiles", out.Tiles), slog.Int("changed", out.Changed), slog.Int("restored", out.Restored),
		slog.Int("painters", out.PaintersTotal),
	}

	if err != nil {
		a.logger.Warn("admin area restore failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin area restore", attrs...)

	return out, nil
}
