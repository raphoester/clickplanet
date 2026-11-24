package stop_renewal_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

type Subscriptions interface {
	StopAtPeriodEnd(ctx context.Context, id renewals.SubscriptionID) error
}

type In struct {
	Account string
	Linked  bool
	Renewal string
}

func New(clock cptime.Clock, store renewals.Store, subscriptions Subscriptions) *UseCase {
	return &UseCase{clock: clock, store: store, subscriptions: subscriptions}
}

type UseCase struct {
	clock         cptime.Clock
	store         renewals.Store
	subscriptions Subscriptions
}

// Execute stops a renewal at the end of the period paid. Stripe first: a renewal kept as stopping never charges. No
// transaction is held over Stripe's call; Stripe's own word of the stop comes later, and the latest word holds.
func (u *UseCase) Execute(ctx context.Context, in In) (renewals.Snapshot, error) {
	buyer, err := orders.BuyerOf(in.Account, in.Linked)
	if err != nil {
		return renewals.Snapshot{}, err //nolint:wrapcheck // the order names it.
	}
	id, err := renewals.ParseSubscriptionID(in.Renewal)
	if err != nil {
		return renewals.Snapshot{}, fmt.Errorf("%w: %w", renewals.ErrUnknownRenewal, err)
	}
	kept, err := u.store.Renewal(ctx, id)
	if err != nil {
		return renewals.Snapshot{}, fmt.Errorf("failed to read the renewal: %w", err)
	}
	stopped, changed, err := kept.StoppedBy(buyer, u.clock.Now())
	if err != nil || !changed {
		return stopped.TakeSnapshot(), err //nolint:wrapcheck // the renewal named it.
	}
	if err := u.subscriptions.StopAtPeriodEnd(ctx, id); err != nil {
		return renewals.Snapshot{}, fmt.Errorf("failed to stop the renewal %s at Stripe: %w", id, err)
	}
	if err := u.store.Save(ctx, stopped); err != nil {
		return renewals.Snapshot{}, fmt.Errorf("failed to keep the renewal %s stopped: %w", id, err)
	}
	return stopped.TakeSnapshot(), nil
}
