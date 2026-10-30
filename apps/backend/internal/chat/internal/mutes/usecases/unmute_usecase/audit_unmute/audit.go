package audit_unmute

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/unmute_usecase"
)

func New(inner unmute_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  unmute_usecase.Executor
	logger *slog.Logger
}

var _ unmute_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context, account messages.AccountID) (time.Time, error) {
	until, err := a.inner.Execute(ctx, account)
	if err != nil {
		a.logger.Warn("admin chat unmute failed", slog.String("account", account.String()), slog.Any("error", err))
		return until, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin chat unmute", slog.String("account", account.String()), slog.Time("mutedUntil", until))

	return until, nil
}
