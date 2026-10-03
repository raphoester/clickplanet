package get_subscription_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

type Accounts interface {
	Account(ctx context.Context, account subscriptions.AccountID) (subscriptions.Account, error)
}

type Store interface {
	Subscription(ctx context.Context, account subscriptions.AccountID) (*subscriptions.Subscription, error)
	Save(ctx context.Context, subscription *subscriptions.Subscription) error
}

type Audience interface {
	Joined(ctx context.Context, address subscriptions.Address) (bool, error)
}

type UseCase struct {
	accounts Accounts
	store    Store
	audience Audience
}

func New(accounts Accounts, store Store, audience Audience) *UseCase {
	return &UseCase{accounts: accounts, store: store, audience: audience}
}

func (u *UseCase) Execute(ctx context.Context, id subscriptions.AccountID) (subscriptions.Status, error) {
	account, err := u.accounts.Account(ctx, id)
	if err != nil {
		return subscriptions.Status{}, fmt.Errorf("failed to read the account: %w", err)
	}

	found, err := u.store.Subscription(ctx, id)
	if errors.Is(err, subscriptions.ErrNoSubscription) {
		return subscriptions.StatusOf(account), nil
	}
	if err != nil {
		return subscriptions.Status{}, fmt.Errorf("failed to read the subscription: %w", err)
	}
	if !found.Waiting() {
		return found.Status(account), nil
	}

	joined, err := u.audience.Joined(ctx, found.Address)
	if err != nil || !joined {
		// A list that cannot answer leaves the row waiting until the next read.
		return found.Status(account), nil
	}

	confirmed := found.Confirmed()
	if err := u.store.Save(ctx, confirmed); err != nil {
		return subscriptions.Status{}, fmt.Errorf("failed to keep the confirmation: %w", err)
	}
	return confirmed.Status(account), nil
}
