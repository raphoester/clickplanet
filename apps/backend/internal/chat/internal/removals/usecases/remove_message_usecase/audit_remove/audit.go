package audit_remove

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_message_usecase"
)

func New(inner remove_message_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  remove_message_usecase.Executor
	logger *slog.Logger
}

var _ remove_message_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context, in remove_message_usecase.In) (time.Time, error) {
	removedAt, err := a.inner.Execute(ctx, in)
	what := []any{slog.String("message", string(in.Message)), slog.String("reason", in.Reason)}
	if err != nil {
		a.logger.Warn("admin chat message removal failed", append(what, slog.Any("error", err))...)
		return removedAt, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin chat message removal", what...)
	return removedAt, nil
}
