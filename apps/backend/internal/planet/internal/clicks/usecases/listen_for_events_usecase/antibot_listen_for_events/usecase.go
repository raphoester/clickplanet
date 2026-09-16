// Package antibot_listen_for_events tells the guard each live stream a caller opens.
package antibot_listen_for_events

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, sink listen_for_events_usecase.Sink) error
}

type ListenGuard interface {
	Listened(scope string)
}

func New(implementation UseCase, guard ListenGuard) *Decorator {
	return &Decorator{implementation: implementation, guard: guard}
}

type Decorator struct {
	implementation UseCase
	guard          ListenGuard
}

func (d *Decorator) Execute(ctx context.Context, sink listen_for_events_usecase.Sink) error {
	d.guard.Listened(cpctx.RateLimitKey(ctx))

	if err := d.implementation.Execute(ctx, sink); err != nil {
		return fmt.Errorf("failed to follow the planet: %w", err)
	}

	return nil
}
