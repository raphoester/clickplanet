package audit_remove_messages

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_messages_usecase"
)

func New(inner remove_messages_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  remove_messages_usecase.Executor
	logger *slog.Logger
}

var _ remove_messages_usecase.Executor = (*Audited)(nil)

// One line per message removed, as audit_remove writes it, so a search for a message finds it however it went.
func (a *Audited) Execute(ctx context.Context, in remove_messages_usecase.In) (time.Time, error) {
	removedAt, err := a.inner.Execute(ctx, in)
	if err != nil {
		ids := make([]string, 0, len(in.Removals))
		for _, removal := range in.Removals {
			ids = append(ids, string(removal.Message))
		}
		a.logger.Warn("admin chat message removals failed", slog.Any("messages", ids), slog.Any("error", err))
		return removedAt, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	for _, removal := range in.Removals {
		a.logger.Warn("admin chat message removal",
			slog.String("message", string(removal.Message)), slog.String("reason", removal.Reason),
			slog.Int("batch", len(in.Removals)))
	}
	return removedAt, nil
}
