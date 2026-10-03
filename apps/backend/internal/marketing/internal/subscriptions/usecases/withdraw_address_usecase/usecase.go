package withdraw_address_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	SubscriptionsTo(ctx context.Context, address subscriptions.Address) ([]*subscriptions.Subscription, error)
	Save(ctx context.Context, subscription *subscriptions.Subscription) error
}

type UseCase struct {
	store Store
	clock cptime.Clock
}

func New(store Store, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, address subscriptions.Address) error {
	found, err := u.store.SubscriptionsTo(ctx, address)
	if err != nil {
		return fmt.Errorf("failed to read the subscriptions to the address: %w", err)
	}

	for _, subscription := range found {
		if !subscription.Live() {
			continue
		}
		if err := u.store.Save(ctx, subscription.Withdrawn(u.clock.Now())); err != nil {
			return fmt.Errorf("failed to keep the withdrawal: %w", err)
		}
	}
	return nil
}
