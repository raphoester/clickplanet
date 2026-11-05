package get_holdings_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Query interface {
	Holdings() *planetv1.GetHoldingsResponse
}

func New(query Query) GetHoldingsHandler {
	return GetHoldingsHandler{query: query}
}

type GetHoldingsHandler struct {
	query Query
}

func (h GetHoldingsHandler) GetHoldings(
	_ context.Context,
	_ *connect.Request[planetv1.GetHoldingsRequest],
) (*connect.Response[planetv1.GetHoldingsResponse], error) {
	return connect.NewResponse(h.query.Holdings()), nil
}
