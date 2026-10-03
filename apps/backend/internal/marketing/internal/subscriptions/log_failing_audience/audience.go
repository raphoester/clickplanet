package log_failing_audience

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

func New(inner subscriptions.Audience, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  subscriptions.Audience
	logger *slog.Logger
}

var _ subscriptions.Audience = (*Logged)(nil)

func (l *Logged) Join(ctx context.Context, address subscriptions.Address) error {
	return l.logged("join", l.inner.Join(ctx, address))
}

func (l *Logged) Invite(ctx context.Context, address subscriptions.Address) error {
	return l.logged("invite", l.inner.Invite(ctx, address))
}

func (l *Logged) Joined(ctx context.Context, address subscriptions.Address) (bool, error) {
	joined, err := l.inner.Joined(ctx, address)
	return joined, l.logged("joined", err)
}

func (l *Logged) Leave(ctx context.Context, address subscriptions.Address) error {
	return l.logged("leave", l.inner.Leave(ctx, address))
}

func (l *Logged) Forget(ctx context.Context, address subscriptions.Address) error {
	return l.logged("forget", l.inner.Forget(ctx, address))
}

func (l *Logged) logged(call string, err error) error {
	if err != nil {
		l.logger.Error("the mailing list refused a call", slog.String("call", call), slog.Any("error", err))
	}
	return err
}
