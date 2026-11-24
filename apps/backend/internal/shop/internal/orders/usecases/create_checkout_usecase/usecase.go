package create_checkout_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/checkouts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

// Payables asks the flags module whether the account may buy time for the target now.
type Payables interface {
	Payable(ctx context.Context, buyer orders.AccountID, target orders.Target) error
}

// Pages are Stripe's Checkout Sessions.
type Pages interface {
	Checkout(ctx context.Context, order orders.Order, price plans.Price) (checkouts.Page, error)
}

type Renewals interface {
	Bought(ctx context.Context, buyer orders.AccountID) ([]renewals.Renewal, error)
}

// Stores is what a checkout reads and keeps: the buyer's renewals, and the pages opened.
type Stores struct {
	Renewals  Renewals
	Checkouts checkouts.Store
}

type In struct {
	Account   string
	Linked    bool
	Plan      string
	Target    orders.Target
	StartsNow bool
}

func New(clock cptime.Clock, catalog plans.Catalog, payables Payables, stores Stores, pages Pages) *UseCase {
	return &UseCase{clock: clock, catalog: catalog, payables: payables, stores: stores, pages: pages}
}

type UseCase struct {
	clock    cptime.Clock
	catalog  plans.Catalog
	payables Payables
	stores   Stores
	pages    Pages
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
	bought, err := u.stores.Renewals.Bought(ctx, buyer)
	if err != nil {
		return "", fmt.Errorf("failed to read the buyer's renewals: %w", err)
	}
	if err := renewals.NoneRenewing(bought, in.Target); err != nil {
		return "", err //nolint:wrapcheck // the renewals name it.
	}
	page, err := u.pages.Checkout(ctx, order, price)
	if err != nil {
		return "", fmt.Errorf("failed to make the checkout: %w", err)
	}
	if err := u.stores.Checkouts.Save(ctx, checkouts.Opened(page, in.Target, u.clock.Now())); err != nil {
		return "", fmt.Errorf("failed to keep the checkout: %w", err)
	}
	return page.URL, nil
}
