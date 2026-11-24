package list_my_renewals_handler

import (
	"context"

	"connectrpc.com/connect"

	shopv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/shop/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
)

type Query interface {
	Mine(ctx context.Context, buyer orders.AccountID) (*shopv1.ListMyRenewalsResponse, error)
}

func New(query Query) ListMyRenewalsHandler {
	return ListMyRenewalsHandler{query: query}
}

type ListMyRenewalsHandler struct {
	query Query
}

func (h ListMyRenewalsHandler) ListMyRenewals(
	ctx context.Context, _ *connect.Request[shopv1.ListMyRenewalsRequest],
) (*connect.Response[shopv1.ListMyRenewalsResponse], error) {
	buyer, err := orders.BuyerOf(cpctx.GetAccount(ctx), cpctx.GetLinked(ctx))
	if err != nil {
		return connect.NewResponse(&shopv1.ListMyRenewalsResponse{}), nil //nolint:nilerr // a guest bought none.
	}
	mine, err := h.query.Mine(ctx, buyer)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(mine), nil
}
