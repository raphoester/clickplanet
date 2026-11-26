package list_plans_handler

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	shopv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/shop/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/renewalmessage"
)

func New(catalog plans.Catalog, clock cptime.Clock) ListPlansHandler {
	prices := catalog.Prices()
	listed := make([]*shopv1.PlanPrice, 0, len(prices))
	for _, price := range prices {
		listed = append(listed, &shopv1.PlanPrice{
			Plan: renewalmessage.Plan(price.Plan), Amount: price.Amount, Currency: price.Currency,
			Renews: price.Plan.Renews(),
		})
	}
	return ListPlansHandler{catalog: catalog, clock: clock, plans: listed}
}

// The plans come from the config, read once at boot; the sale is told by the server's clock at each call.
type ListPlansHandler struct {
	catalog plans.Catalog
	clock   cptime.Clock
	plans   []*shopv1.PlanPrice
}

func (h ListPlansHandler) ListPlans(
	context.Context, *connect.Request[shopv1.ListPlansRequest],
) (*connect.Response[shopv1.ListPlansResponse], error) {
	answer := &shopv1.ListPlansResponse{Plans: h.plans}
	if sale, runs := h.catalog.SaleAt(h.clock.Now()); runs {
		answer.Sale = &shopv1.Sale{PercentOff: sale.PercentOff, EndsAt: timestamppb.New(sale.Until)}
	}
	return connect.NewResponse(answer), nil
}
