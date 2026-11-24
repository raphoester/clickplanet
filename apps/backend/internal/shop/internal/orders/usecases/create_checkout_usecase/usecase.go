package create_checkout_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

// Payables asks the flags module whether the account may buy time for the target now.
type Payables interface {
	Payable(ctx context.Context, buyer orders.AccountID, target orders.Target) error
}

type Checkouts interface {
	Checkout(ctx context.Context, order orders.Order, price plans.Price) (string, error)
}

type Renewals interface {
	Bought(ctx context.Context, buyer orders.AccountID) ([]renewals.Renewal, error)
}

type In struct {
	Account   string
	Linked    bool
	Plan      string
	Target    orders.Target
	StartsNow bool
}

func New(
	clock cptime.Clock, catalog plans.Catalog, payables Payables, renewals Renewals, checkouts Checkouts,
) *UseCase {
	return &UseCase{clock: clock, catalog: catalog, payables: payables, renewals: renewals, checkouts: checkouts}
}

type UseCase struct {
	clock     cptime.Clock
	catalog   plans.Catalog
	payables  Payables
	renewals  Renewals
	checkouts Checkouts
}

// Execute answers Stripe's page for the order.
func (u *UseCase) Execute(ctx context.Context, in In) (string, error) {
	buyer, err := orders.BuyerOf(in.Account, in.Linked)
	if err != nil {
		return "", err //nolint:wrapcheck // the order names it.
	}
	plan, err := plans.ParsePlan(in.Plan)
	if err != nil {
		return "", err //nolint:wrapcheck // the parse names it.
	}
	price, err := u.catalog.Price(plan)
	if err != nil {
		return "", err //nolint:wrapcheck // the catalog names it.
	}
	order, err := orders.Place(buyer, plan, in.Target, in.StartsNow, u.clock.Now())
	if err != nil {
		return "", err //nolint:wrapcheck // the order names it.
	}
	if err := u.payables.Payable(ctx, buyer, in.Target); err != nil {
		return "", fmt.Errorf("the flags module refused the order: %w", err)
	}
	bought, err := u.renewals.Bought(ctx, buyer)
	if err != nil {
		return "", fmt.Errorf("failed to read the buyer's renewals: %w", err)
	}
	if err := renewals.NoneRenewing(bought, in.Target); err != nil {
		return "", err //nolint:wrapcheck // the renewals name it.
	}
	page, err := u.checkouts.Checkout(ctx, order, price)
	if err != nil {
		return "", fmt.Errorf("failed to make the checkout: %w", err)
	}
	return page, nil
}
