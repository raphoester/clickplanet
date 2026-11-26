package plans_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
)

var (
	starts = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	ends   = time.Date(2026, 11, 30, 22, 59, 0, 0, time.UTC)
)

func blackFriday() plans.SaleSnapshot {
	return plans.SaleSnapshot{Coupon: "BLACKFRIDAY2026", PercentOff: 50, From: starts, Until: ends}
}

func TestASaleRunsFromItsStartUntilItsEnd(t *testing.T) {
	sale, err := blackFriday().Restore()
	require.NoError(t, err)

	assert.False(t, sale.Runs(starts.Add(-time.Second)))
	assert.True(t, sale.Runs(starts))
	assert.True(t, sale.Runs(ends.Add(-time.Second)))
	assert.False(t, sale.Runs(ends), "its end is the coupon's redeem_by")
	assert.Equal(t, []plans.CouponID{"BLACKFRIDAY2026"}, sale.CouponsAt(starts))
	assert.Empty(t, sale.CouponsAt(ends))
	assert.Equal(t, blackFriday(), sale.TakeSnapshot())
}

func TestTheZeroSaleNeverRuns(t *testing.T) {
	assert.False(t, plans.Sale{}.Runs(starts))
	assert.Empty(t, plans.Sale{}.CouponsAt(starts))
}

func TestASaleOfNoCouponNoPercentOrNoTimeIsRefused(t *testing.T) {
	for name, change := range map[string]func(*plans.SaleSnapshot){
		"no coupon":         func(s *plans.SaleSnapshot) { s.Coupon = "" },
		"a coupon of space": func(s *plans.SaleSnapshot) { s.Coupon = "BLACK FRIDAY" },
		"nothing off":       func(s *plans.SaleSnapshot) { s.PercentOff = 0 },
		"all off":           func(s *plans.SaleSnapshot) { s.PercentOff = 100 },
		"no start":          func(s *plans.SaleSnapshot) { s.From = time.Time{} },
		"an end before":     func(s *plans.SaleSnapshot) { s.Until = starts.Add(-time.Hour) },
		"no time at all":    func(s *plans.SaleSnapshot) { s.Until = starts },
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := blackFriday()
			change(&snapshot)

			_, err := snapshot.Restore()

			require.ErrorIs(t, err, plans.ErrInvalidSale)
		})
	}
}

func TestTheCatalogTellsTheSaleOnlyWhileItRuns(t *testing.T) {
	sale, err := blackFriday().Restore()
	require.NoError(t, err)
	catalog, err := plans.NewCatalog(
		sale,
		plans.Price{Plan: plans.Week, ID: "price_week", Amount: 400, Currency: "eur"},
		plans.Price{Plan: plans.Month, ID: "price_month", Amount: 1000, Currency: "eur"},
		plans.Price{Plan: plans.Year, ID: "price_year", Amount: 6000, Currency: "eur"},
	)
	require.NoError(t, err)

	told, runs := catalog.SaleAt(starts)
	assert.True(t, runs)
	assert.Equal(t, blackFriday(), told)
	_, runs = catalog.SaleAt(ends)
	assert.False(t, runs)
	offer, err := catalog.OfferAt(plans.Month, starts)
	require.NoError(t, err)
	assert.Equal(t, []plans.CouponID{"BLACKFRIDAY2026"}, offer.Coupons)
}

func TestTheShopOffHasNoSale(t *testing.T) {
	sale, err := blackFriday().Restore()
	require.NoError(t, err)
	catalog, err := plans.NewCatalog(sale)
	require.NoError(t, err)

	_, runs := catalog.SaleAt(starts)

	assert.False(t, runs)
}
