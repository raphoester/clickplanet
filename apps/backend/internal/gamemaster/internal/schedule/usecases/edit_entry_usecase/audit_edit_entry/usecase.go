package audit_edit_entry

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type UseCase interface {
	Execute(ctx context.Context, id schedule.EntryID, draft schedule.Draft) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, id schedule.EntryID, draft schedule.Draft) error {
	err := a.inner.Execute(ctx, id, draft)

	attrs := append(attrsOf(draft), slog.String("entry", id.String()))
	if err != nil {
		a.logger.Warn("admin schedule edit refused", append(attrs, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin schedule entry edited", attrs...)
	return nil
}

func attrsOf(draft schedule.Draft) []any {
	return []any{
		slog.String("set", string(draft.Set)),
		slog.String("kind", string(draft.Kind)),
		slog.String("description", string(draft.Description)),
		slog.String("starts_at", draft.StartsAt.UTC().Format(time.RFC3339)),
		slog.String("ends_at", draft.EndsAt.UTC().Format(time.RFC3339)),
		slog.Uint64("priority", uint64(draft.Priority)),
	}
}
