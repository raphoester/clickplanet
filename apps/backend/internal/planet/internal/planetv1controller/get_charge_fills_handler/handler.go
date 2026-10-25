package get_charge_fills_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Query interface {
	ChargeFills(ctx context.Context) (*planetv1.GetChargeFillsResponse, error)
}

func New(query Query) GetChargeFillsHandler {
	return GetChargeFillsHandler{query: query}
}

type GetChargeFillsHandler struct {
	query Query
}

func (h GetChargeFillsHandler) GetChargeFills(
	ctx context.Context,
	_ *connect.Request[planetv1.GetChargeFillsRequest],
) (*connect.Response[planetv1.GetChargeFillsResponse], error) {
	answer, err := h.query.ChargeFills(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the query says what failed.
	}

	return connect.NewResponse(answer), nil
}
