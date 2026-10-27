package audit_add_entry

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type UseCase interface {
	Execute(ctx context.Context, draft schedule.Draft) (schedule.EntryID, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, draft schedule.Draft) (schedule.EntryID, error) {
	id, err := a.inner.Execute(ctx, draft)
	if err != nil {
		a.logger.Warn("admin schedule entry refused", append(attrsOf(draft), slog.Any("error", err))...)
		return id, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin schedule entry added", append(attrsOf(draft), slog.String("entry", id.String()))...)
	return id, nil
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
