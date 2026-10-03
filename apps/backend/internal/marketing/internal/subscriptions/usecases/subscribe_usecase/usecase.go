package subscribe_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Accounts interface {
	Account(ctx context.Context, account subscriptions.AccountID) (subscriptions.Account, error)
}

type Store interface {
	Subscription(ctx context.Context, account subscriptions.AccountID) (*subscriptions.Subscription, error)
	Save(ctx context.Context, subscription *subscriptions.Subscription) error
}

type Audience interface {
	Join(ctx context.Context, address subscriptions.Address) error
	Invite(ctx context.Context, address subscriptions.Address) error
	Forget(ctx context.Context, address subscriptions.Address) error
}

type UseCase struct {
	accounts Accounts
	store    Store
	audience Audience
	clock    cptime.Clock
}

func New(accounts Accounts, store Store, audience Audience, clock cptime.Clock) *UseCase {
	return &UseCase{accounts: accounts, store: store, audience: audience, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, id subscriptions.AccountID, raw string) (subscriptions.State, error) {
	address, err := subscriptions.AddressOf(raw)
	if err != nil {
		return "", err //nolint:wrapcheck // the handler maps the sentinel.
	}

	account, err := u.accounts.Account(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to read the account: %w", err)
	}
	subscription, err := subscriptions.NewSubscription(account, address, u.clock.Now())
	if err != nil {
		return "", err //nolint:wrapcheck // the handler maps the sentinel.
	}

	existing, err := u.store.Subscription(ctx, id)
	switch {
	case errors.Is(err, subscriptions.ErrNoSubscription):
	case err != nil:
		return "", fmt.Errorf("failed to read the subscription: %w", err)
	default:
		if err := existing.ChangeError(address); err != nil {
			return "", err //nolint:wrapcheck // the handler maps the sentinel.
		}
	}

	if err := u.ask(ctx, subscription); err != nil {
		return "", fmt.Errorf("%w: %w", subscriptions.ErrAudienceUnreachable, err)
	}
	if err := u.store.Save(ctx, subscription); err != nil {
		return "", errors.Join(fmt.Errorf("failed to keep the subscription: %w", err), u.audience.Forget(ctx, address))
	}
	return subscription.State, nil
}

func (u *UseCase) ask(ctx context.Context, subscription *subscriptions.Subscription) error {
	if subscription.Waiting() {
		return u.audience.Invite(ctx, subscription.Address) //nolint:wrapcheck // the caller names it.
	}
	return u.audience.Join(ctx, subscription.Address) //nolint:wrapcheck // the caller names it.
}
