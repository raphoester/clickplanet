package forget_account_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

type Store interface {
	Subscription(ctx context.Context, account subscriptions.AccountID) (*subscriptions.Subscription, error)
	Delete(ctx context.Context, account subscriptions.AccountID) error
}

type Audience interface {
	Forget(ctx context.Context, address subscriptions.Address) error
}

type UseCase struct {
	store    Store
	audience Audience
}

func New(store Store, audience Audience) *UseCase {
	return &UseCase{store: store, audience: audience}
}

func (u *UseCase) Execute(ctx context.Context, id subscriptions.AccountID) error {
	found, err := u.store.Subscription(ctx, id)
	if errors.Is(err, subscriptions.ErrNoSubscription) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the subscription: %w", err)
	}

	if err := u.audience.Forget(ctx, found.Address); err != nil {
		return fmt.Errorf("failed to forget the contact: %w", err)
	}
	if err := u.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete the subscription: %w", err)
	}
	return nil
}
