package create_checkout_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/checkouts"
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

// payables answers the target asked, and the other targets of its flag.
type payables struct {
	refusal error
	others  []orders.Target
}

func (p payables) Payable(_ context.Context, _ orders.AccountID, target orders.Target) ([]orders.Target, error) {
	if p.refusal != nil {
		return nil, p.refusal
	}
	return append([]orders.Target{target}, p.others...), nil
}

type pages struct {
	orders []orders.Snapshot
	offers []plans.Offer
}

func (p *pages) Checkout(_ context.Context, order orders.Order, offer plans.Offer) (checkouts.Page, error) {
	p.orders = append(p.orders, order.TakeSnapshot())
	p.offers = append(p.offers, offer)
	return checkouts.Page{Session: "cs_1", URL: "https://checkout.stripe.com/c/pay/cs_1"}, nil
}

func stores(kept *renewals.MemoryStore) create_checkout_usecase.Stores {
	return create_checkout_usecase.Stores{Renewals: kept, Checkouts: checkouts.NewMemoryStore()}
}

var (
	saleStarts = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	saleEnds   = time.Date(2026, 11, 30, 22, 59, 0, 0, time.UTC)
)

func catalog(t *testing.T) plans.Catalog {
	t.Helper()
	return catalogOnSale(t, plans.Sale{})
}

func blackFriday(t *testing.T) plans.Sale {
	t.Helper()
	sale, err := plans.SaleSnapshot{Coupon: "BLACKFRIDAY2026", PercentOff: 50, From: saleStarts, Until: saleEnds}.Restore()
	require.NoError(t, err)
	return sale
}

func catalogOnSale(t *testing.T, sale plans.Sale) plans.Catalog {
	t.Helper()
	priced, err := plans.NewCatalog(
		sale,
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
	sold, opened := &pages{}, checkouts.NewMemoryStore()
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t), payables{},
		create_checkout_usecase.Stores{Renewals: renewals.NewMemoryStore(), Checkouts: opened}, sold)

	page, err := useCase.Execute(t.Context(), in(t, "month"))
	require.NoError(t, err)

	assert.Equal(t, "https://checkout.stripe.com/c/pay/cs_1", page)
	assert.Equal(t, []orders.Snapshot{{Buyer: buyer, Plan: plans.Month, Target: target(t), AgreedAt: now}}, sold.orders)
	assert.Equal(t, []plans.Offer{{
		Price: plans.Price{Plan: plans.Month, ID: "price_month", Amount: 1000, Currency: "eur"}, Coupons: []plans.CouponID{},
	}}, sold.offers, "no sale, the full price")
	kept, err := opened.OfTarget(t.Context(), target(t))
	require.NoError(t, err)
	require.Len(t, kept, 1, "kept, so a rejection closes the page")
	assert.Equal(t, checkouts.Snapshot{Session: "cs_1", Target: target(t), OpenedAt: now}, kept[0].TakeSnapshot())
}

func TestNoPageIsMadeForWhatTheFlagsModuleRefuses(t *testing.T) {
	sold := &pages{}
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t),
		payables{refusal: rpc_payables.ErrNotYours}, stores(renewals.NewMemoryStore()), sold)

	_, err := useCase.Execute(t.Context(), in(t, "week"))

	require.ErrorIs(t, err, rpc_payables.ErrNotYours)
	assert.Empty(t, sold.orders)
}

func TestAGuestBuysNothing(t *testing.T) {
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t), payables{}, stores(renewals.NewMemoryStore()),
		&pages{})
	guest := in(t, "week")
	guest.Linked = false

	_, err := useCase.Execute(t.Context(), guest)

	require.ErrorIs(t, err, orders.ErrGuest)
}

func TestNoPageWithoutTheBuyersAgreementToStartNow(t *testing.T) {
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t), payables{}, stores(renewals.NewMemoryStore()),
		&pages{})
	unagreed := in(t, "week")
	unagreed.StartsNow = false

	_, err := useCase.Execute(t.Context(), unagreed)

	require.ErrorIs(t, err, orders.ErrNoAgreement)
}

func TestTheShopOffSellsNothing(t *testing.T) {
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), plans.Catalog{}, payables{},
		stores(renewals.NewMemoryStore()), &pages{})

	_, err := useCase.Execute(t.Context(), in(t, "week"))

	require.ErrorIs(t, err, plans.ErrOff)
}

func TestASecondPlanIsNotSoldWhileARenewalRuns(t *testing.T) {
	kept := renewals.NewMemoryStore()
	kept.Seed(renewals.TestRenewing("sub_1", buyer, now.AddDate(0, 1, 0), now))
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t), payables{}, stores(kept), &pages{})

	_, err := useCase.Execute(t.Context(), in(t, "year"))

	require.ErrorIs(t, err, renewals.ErrRenewing)
}

// A flag's holder renews it from the page that opened it: its renewal names the submission, and the flag is bought on
// its code.
func TestNoPlayerClaimsAFlagWhileItsHoldersRenewalRunsWhateverTargetItNames(t *testing.T) {
	kept := renewals.NewMemoryStore()
	kept.Seed(renewals.TestRenewing("sub_1", buyer, now.AddDate(0, 1, 0), now))
	flag, err := orders.ParseTarget("", "r-0000000001")
	require.NoError(t, err)
	sold := &pages{}
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t),
		payables{others: []orders.Target{target(t)}}, stores(kept), sold)
	claim := in(t, "week")
	claim.Account, claim.Target = orders.AccountID{15: 2}.String(), flag

	_, err = useCase.Execute(t.Context(), claim)

	require.ErrorIs(t, err, renewals.ErrRenewing, "Stripe may still retry the holder's card")
	assert.Empty(t, sold.orders)
}

func TestAPlayerClaimsAFlagOnceItsHoldersRenewalStopped(t *testing.T) {
	kept := renewals.NewMemoryStore()
	stopping := renewals.TestRenewing("sub_1", buyer, now.AddDate(0, 1, 0), now)
	stopping.State = renewals.Ended
	kept.Seed(stopping)
	flag, err := orders.ParseTarget("", "r-0000000001")
	require.NoError(t, err)
	useCase := create_checkout_usecase.New(cptime.NewFixedClock(now), catalog(t),
		payables{others: []orders.Target{target(t)}}, stores(kept), &pages{})
	claim := in(t, "week")
	claim.Account, claim.Target = orders.AccountID{15: 2}.String(), flag

	_, err = useCase.Execute(t.Context(), claim)

	require.NoError(t, err)
}

func TestTheSalesCouponGoesToStripeWhileTheSaleRuns(t *testing.T) {
	for name, at := range map[string]time.Time{
		"at its start":            saleStarts,
		"during it":               saleStarts.Add(50 * time.Hour),
		"a second before its end": saleEnds.Add(-time.Second),
	} {
		t.Run(name, func(t *testing.T) {
			sold := &pages{}
			useCase := create_checkout_usecase.New(cptime.NewFixedClock(at), catalogOnSale(t, blackFriday(t)), payables{},
				stores(renewals.NewMemoryStore()), sold)

			_, err := useCase.Execute(t.Context(), in(t, "year"))
			require.NoError(t, err)

			require.Len(t, sold.offers, 1)
			assert.Equal(t, []plans.CouponID{"BLACKFRIDAY2026"}, sold.offers[0].Coupons)
			assert.Equal(t, plans.Price{Plan: plans.Year, ID: "price_year", Amount: 6000, Currency: "eur"},
				sold.offers[0].Price, "the plan's own price: Stripe takes the coupon off it")
		})
	}
}

func TestNoCouponGoesToStripeOutsideTheSale(t *testing.T) {
	for name, at := range map[string]time.Time{
		"a second before its start": saleStarts.Add(-time.Second),
		"at its end":                saleEnds,
		"after it":                  saleEnds.Add(time.Hour),
	} {
		t.Run(name, func(t *testing.T) {
			sold := &pages{}
			useCase := create_checkout_usecase.New(cptime.NewFixedClock(at), catalogOnSale(t, blackFriday(t)), payables{},
				stores(renewals.NewMemoryStore()), sold)

			_, err := useCase.Execute(t.Context(), in(t, "week"))
			require.NoError(t, err)

			require.Len(t, sold.offers, 1)
			assert.Empty(t, sold.offers[0].Coupons)
		})
	}
}
