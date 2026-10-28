package audit_edit_season

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type UseCase interface {
	Execute(ctx context.Context, number season.Number, draft season.Draft) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, number season.Number, draft season.Draft) error {
	err := a.inner.Execute(ctx, number, draft)

	attrs := []any{
		slog.Uint64("season", uint64(number)),
		slog.Duration("break", draft.Break),
		slog.Duration("length", draft.Length),
	}
	if err != nil {
		a.logger.Warn("admin season edit refused", append(attrs, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin season edited", attrs...)
	return nil
}
