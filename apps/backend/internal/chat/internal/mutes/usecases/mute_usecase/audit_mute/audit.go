package audit_mute

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
)

func New(inner mute_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  mute_usecase.Executor
	logger *slog.Logger
}

var _ mute_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context, in mute_usecase.In) (mutes.Mute, error) {
	mute, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("account", in.Account.String()), slog.Duration("duration", in.Duration),
		slog.String("scope", string(mute.Caller().Scope())), slog.Time("mutedUntil", mute.Until()),
	}

	if err != nil {
		a.logger.Warn("admin chat mute failed", append(attrs, slog.Any("error", err))...)
		return mute, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin chat mute", attrs...)

	return mute, nil
}
