package list_plans_handler

import (
	"context"

	"connectrpc.com/connect"

	shopv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/shop/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/renewalmessage"
)

func New(catalog plans.Catalog) ListPlansHandler {
	prices := catalog.Prices()
	listed := make([]*shopv1.PlanPrice, 0, len(prices))
	for _, price := range prices {
		listed = append(listed, &shopv1.PlanPrice{
			Plan: renewalmessage.Plan(price.Plan), Amount: price.Amount, Currency: price.Currency,
			Renews: price.Plan.Renews(),
		})
	}
	return ListPlansHandler{plans: listed}
}

// The plans come from the config, read once at boot.
type ListPlansHandler struct {
	plans []*shopv1.PlanPrice
}

func (h ListPlansHandler) ListPlans(
	context.Context, *connect.Request[shopv1.ListPlansRequest],
) (*connect.Response[shopv1.ListPlansResponse], error) {
	return connect.NewResponse(&shopv1.ListPlansResponse{Plans: h.plans}), nil
}
