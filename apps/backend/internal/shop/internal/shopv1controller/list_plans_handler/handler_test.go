package list_plans_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	shopv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/shop/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/list_plans_handler"
)

var (
	starts = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	ends   = time.Date(2026, 11, 30, 22, 59, 0, 0, time.UTC)
)

func catalog(t *testing.T, sale plans.Sale) plans.Catalog {
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

func blackFriday(t *testing.T) plans.Sale {
	t.Helper()
	sale, err := plans.SaleSnapshot{Coupon: "BLACKFRIDAY2026", PercentOff: 50, From: starts, Until: ends}.Restore()
	require.NoError(t, err)
	return sale
}

func listed(t *testing.T, catalog plans.Catalog, at time.Time) *shopv1.ListPlansResponse {
	t.Helper()
	answer, err := list_plans_handler.New(catalog, cptime.NewFixedClock(at)).
		ListPlans(t.Context(), connect.NewRequest(&shopv1.ListPlansRequest{}))
	require.NoError(t, err)
	return answer.Msg
}

var fullPrices = []*shopv1.PlanPrice{
	{Plan: shopv1.Plan_PLAN_WEEK, Amount: 400, Currency: "eur"},
	{Plan: shopv1.Plan_PLAN_MONTH, Amount: 1000, Currency: "eur", Renews: true},
	{Plan: shopv1.Plan_PLAN_YEAR, Amount: 6000, Currency: "eur", Renews: true},
}

func TestThePlansComeWithNoSaleWhenNoneIsSet(t *testing.T) {
	answer := listed(t, catalog(t, plans.Sale{}), starts)

	want := &shopv1.ListPlansResponse{Plans: fullPrices}
	assert.True(t, proto.Equal(want, answer), "got %v", answer)
}

func TestThePlansComeWithTheSaleWhileItRuns(t *testing.T) {
	for name, at := range map[string]time.Time{
		"at its start":            starts,
		"a second before its end": ends.Add(-time.Second),
	} {
		t.Run(name, func(t *testing.T) {
			answer := listed(t, catalog(t, blackFriday(t)), at)

			want := &shopv1.ListPlansResponse{
				Plans: fullPrices, Sale: &shopv1.Sale{PercentOff: 50, EndsAt: timestamppb.New(ends)},
			}
			assert.True(t, proto.Equal(want, answer), "the full prices, the page takes the sale off them; got %v", answer)
		})
	}
}

func TestThePlansComeWithNoSaleOutsideIt(t *testing.T) {
	for name, at := range map[string]time.Time{
		"a second before its start": starts.Add(-time.Second),
		"at its end":                ends,
	} {
		t.Run(name, func(t *testing.T) {
			answer := listed(t, catalog(t, blackFriday(t)), at)

			assert.Nil(t, answer.GetSale())
			assert.Len(t, answer.GetPlans(), 3)
		})
	}
}
