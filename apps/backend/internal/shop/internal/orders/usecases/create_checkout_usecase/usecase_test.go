package create_checkout_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders/usecases/create_checkout_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/rpc_payables"
)

var (
	now   = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	buyer = orders.AccountID{15: 1}
)

type payables struct{ refusal error }

func (p payables) Payable(context.Context, orders.AccountID, orders.Target) error { return p.refusal }

type checkouts struct{ orders []orders.Snapshot }

func (c *checkouts) Checkout(_ context.Context, order orders.Order, _ plans.Price) (string, error) {
	c.orders = append(c.orders, order.TakeSnapshot())
	return "https://checkout.stripe.com/c/pay/cs_1", nil
}

func catalog(t *testing.T) plans.Catalog {
	t.Helper()
	priced, err := plans.NewCatalog(
		plans.Price{Plan: plans.Week, ID: "price_week", Amount: 400, Currency: "eur"},
		plans.Price{Plan: plans.Month, ID: "price_month", Amount: 1000, Currency: "eur"},
		plans.Price{Plan: plans.Year, ID: "price_year", Amount: 6000, Currency: "eur"},
	)
	require.NoError(t, err)
	return priced
}

func target(t *testing.T) orders.Target {
	t.Helper()
	parsed, err := orders.ParseTarget("00000000-0000-0000-0000-000000000001", "")
	require.NoError(t, err)
	return parsed
}

func in(t *testing.T, plan string) create_checkout_usecase.In {
	t.Helper()
	return create_checkout_usecase.In{
		Account: buyer.String(), Linked: true, Plan: plan, Target: target(t), StartsNow: true,
	}
}

func TestTheFounderGetsStripesPageForItsOrder(t *testing.T) {
	sold := &checkouts{}
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t), payables{}, renewals.NewMemoryStore(), sold)

	page, err := useCase.Execute(t.Context(), in(t, "month"))
	require.NoError(t, err)

	assert.Equal(t, "https://checkout.stripe.com/c/pay/cs_1", page)
	assert.Equal(t, []orders.Snapshot{{Buyer: buyer, Plan: plans.Month, Target: target(t), AgreedAt: now}}, sold.orders)
}

func TestNoPageIsMadeForWhatTheFlagsModuleRefuses(t *testing.T) {
	sold := &checkouts{}
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t),
		payables{refusal: rpc_payables.ErrNotYours}, renewals.NewMemoryStore(), sold)

	_, err := useCase.Execute(t.Context(), in(t, "week"))

	require.ErrorIs(t, err, rpc_payables.ErrNotYours)
	assert.Empty(t, sold.orders)
}

func TestAGuestBuysNothing(t *testing.T) {
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t), payables{}, renewals.NewMemoryStore(),
		&checkouts{})
	guest := in(t, "week")
	guest.Linked = false

	_, err := useCase.Execute(t.Context(), guest)

	require.ErrorIs(t, err, orders.ErrGuest)
}

func TestNoPageWithoutTheBuyersAgreementToStartNow(t *testing.T) {
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t), payables{}, renewals.NewMemoryStore(),
		&checkouts{})
	unagreed := in(t, "week")
	unagreed.StartsNow = false

	_, err := useCase.Execute(t.Context(), unagreed)

	require.ErrorIs(t, err, orders.ErrNoAgreement)
}

func TestTheShopOffSellsNothing(t *testing.T) {
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), plans.Catalog{}, payables{},
		renewals.NewMemoryStore(), &checkouts{})

	_, err := useCase.Execute(t.Context(), in(t, "week"))

	require.ErrorIs(t, err, plans.ErrOff)
}

func TestASecondPlanIsNotSoldWhileARenewalRuns(t *testing.T) {
	kept := renewals.NewMemoryStore()
	kept.Seed(renewals.TestRenewing("sub_1", buyer, now.AddDate(0, 1, 0), now))
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t), payables{}, kept, &checkouts{})

	_, err := useCase.Execute(t.Context(), in(t, "year"))

	require.ErrorIs(t, err, renewals.ErrRenewing)
}
