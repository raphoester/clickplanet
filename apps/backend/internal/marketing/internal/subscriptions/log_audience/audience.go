package log_audience

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

type Audience struct {
	logger *slog.Logger
}

var _ subscriptions.Audience = (*Audience)(nil)

func New(logger *slog.Logger) *Audience {
	return &Audience{logger: logger}
}

func (a *Audience) Join(_ context.Context, address subscriptions.Address) error {
	a.logged("join", address)
	return nil
}

func (a *Audience) Invite(_ context.Context, address subscriptions.Address) error {
	a.logged("invite", address)
	return nil
}

func (a *Audience) Joined(_ context.Context, address subscriptions.Address) (bool, error) {
	a.logged("joined", address)
	return true, nil
}

func (a *Audience) Leave(_ context.Context, address subscriptions.Address) error {
	a.logged("leave", address)
	return nil
}

func (a *Audience) Forget(_ context.Context, address subscriptions.Address) error {
	a.logged("forget", address)
	return nil
}

func (a *Audience) logged(call string, address subscriptions.Address) {
	a.logger.Warn("a mailing list call was logged, not sent", slog.String("call", call), slog.String("address", string(address)))
}
