package audit_delete_entry

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule"
)

type UseCase interface {
	Execute(ctx context.Context, id schedule.EntryID) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, id schedule.EntryID) error {
	err := a.inner.Execute(ctx, id)
	if err != nil {
		a.logger.Warn("admin schedule delete refused", slog.String("entry", id.String()), slog.Any("error", err))
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin schedule entry deleted", slog.String("entry", id.String()))
	return nil
}
