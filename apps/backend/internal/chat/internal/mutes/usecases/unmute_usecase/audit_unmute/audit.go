package audit_unmute

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
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

func (a *Audited) Execute(ctx context.Context, account messages.AccountID, kind mutes.Kind) (time.Time, error) {
	until, err := a.inner.Execute(ctx, account, kind)
	who := []any{slog.String("account", account.String()), slog.String("kind", kind.String())}
	if err != nil {
		a.logger.Warn("admin chat unmute failed", append(who, slog.Any("error", err))...)
		return until, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin chat unmute", append(who, slog.Time("mutedUntil", until))...)

	return until, nil
}
