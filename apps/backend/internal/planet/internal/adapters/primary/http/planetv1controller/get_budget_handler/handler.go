// Package get_budget_handler serves planet.v1.ClickService/GetBudget.
package get_budget_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/clickbudget"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/toll"
)

type UseCase interface {
	Execute(ctx context.Context, country string) (toll.Budget, bool)
}

func New(useCase UseCase) GetBudgetHandler {
	return GetBudgetHandler{useCase: useCase}
}

type GetBudgetHandler struct {
	useCase UseCase
}

// GetBudget is how a client that has just loaded learns its allowance. Every
// click answers with a fresh one afterwards, so this is asked once per page load.
func (h GetBudgetHandler) GetBudget(
	ctx context.Context,
	req *connect.Request[planetv1.GetBudgetRequest],
) (*connect.Response[planetv1.GetBudgetResponse], error) {
	res := &planetv1.GetBudgetResponse{}
	if budget, limited := h.useCase.Execute(ctx, req.Msg.GetCountryId()); limited {
		res.Budget = clickbudget.Encode(budget)
	}

	return connect.NewResponse(res), nil
}
