package audit_delete_season

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type UseCase interface {
	Execute(ctx context.Context, number season.Number) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, number season.Number) error {
	err := a.inner.Execute(ctx, number)
	if err != nil {
		a.logger.Warn("admin season delete refused", slog.Uint64("season", uint64(number)), slog.Any("error", err))
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin season deleted", slog.Uint64("season", uint64(number)))
	return nil
}
