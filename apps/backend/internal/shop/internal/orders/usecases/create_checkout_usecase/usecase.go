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

// Payables asks the flags module whether the account may buy time for the target now, and which targets buy time for
// the same flag.
type Payables interface {
	Payable(ctx context.Context, buyer orders.AccountID, target orders.Target) ([]orders.Target, error)
}

// Pages are Stripe's Checkout Sessions.
type Pages interface {
	Checkout(ctx context.Context, order orders.Order, offer plans.Offer) (checkouts.Page, error)
}

type Renewals interface {
	OfTarget(ctx context.Context, target orders.Target) ([]renewals.Renewal, error)
}

// Stores is what a checkout reads and keeps: the renewals of the flag, and the pages opened.
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

// Execute answers Stripe's page for the order, at the sale's price while a sale runs.
func (u *UseCase) Execute(ctx context.Context, in In) (string, error) {
	now := u.clock.Now()
	buyer, err := orders.BuyerOf(in.Account, in.Linked)
	if err != nil {
		return "", err //nolint:wrapcheck // the order names it.
	}
	plan, err := plans.ParsePlan(in.Plan)
	if err != nil {
		return "", err //nolint:wrapcheck // the parse names it.
	}
	offer, err := u.catalog.OfferAt(plan, now)
	if err != nil {
		return "", err //nolint:wrapcheck // the catalog names it.
	}
	order, err := orders.Place(buyer, plan, in.Target, in.StartsNow, now)
	if err != nil {
		return "", err //nolint:wrapcheck // the order names it.
	}
	same, err := u.payables.Payable(ctx, buyer, in.Target)
	if err != nil {
		return "", fmt.Errorf("the flags module refused the order: %w", err)
	}
	var renewing []renewals.Renewal
	for _, target := range same {
		kept, err := u.stores.Renewals.OfTarget(ctx, target)
		if err != nil {
			return "", fmt.Errorf("failed to read the renewals of %v: %w", target, err)
		}
		renewing = append(renewing, kept...)
	}
	if err := renewals.NoneRenewing(renewing); err != nil {
		return "", err //nolint:wrapcheck // the renewals name it.
	}
	page, err := u.pages.Checkout(ctx, order, offer)
	if err != nil {
		return "", fmt.Errorf("failed to make the checkout: %w", err)
	}
	if err := u.stores.Checkouts.Save(ctx, checkouts.Opened(page, in.Target, now)); err != nil {
		return "", fmt.Errorf("failed to keep the checkout: %w", err)
	}
	return page.URL, nil
}
