package unsubscribe_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	Subscription(ctx context.Context, account subscriptions.AccountID) (*subscriptions.Subscription, error)
	Save(ctx context.Context, subscription *subscriptions.Subscription) error
}

type Audience interface {
	Leave(ctx context.Context, address subscriptions.Address) error
}

type UseCase struct {
	store    Store
	audience Audience
	clock    cptime.Clock
}

func New(store Store, audience Audience, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, audience: audience, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, id subscriptions.AccountID) error {
	found, err := u.store.Subscription(ctx, id)
	if errors.Is(err, subscriptions.ErrNoSubscription) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the subscription: %w", err)
	}
	if !found.Live() {
		return nil
	}

	if err := u.audience.Leave(ctx, found.Address); err != nil {
		return fmt.Errorf("%w: %w", subscriptions.ErrAudienceUnreachable, err)
	}
	if err := u.store.Save(ctx, found.Withdrawn(u.clock.Now())); err != nil {
		return fmt.Errorf("failed to keep the withdrawal: %w", err)
	}
	return nil
}
