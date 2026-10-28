package audit_add_season

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type UseCase interface {
	Execute(ctx context.Context, draft season.Draft) (season.Number, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, draft season.Draft) (season.Number, error) {
	number, err := a.inner.Execute(ctx, draft)
	if err != nil {
		a.logger.Warn("admin season refused", append(attrsOf(draft), slog.Any("error", err))...)
		return number, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin season added", append(attrsOf(draft), slog.Uint64("season", uint64(number)))...)
	return number, nil
}

func attrsOf(draft season.Draft) []any {
	return []any{
		slog.Duration("break", draft.Break),
		slog.Duration("length", draft.Length),
	}
}
