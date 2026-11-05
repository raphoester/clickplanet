package get_allegiance_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Query interface {
	Allegiance(ctx context.Context, account allegiances.Account) (*planetv1.GetAllegianceResponse, error)
}

func New(query Query) GetAllegianceHandler {
	return GetAllegianceHandler{query: query}
}

type GetAllegianceHandler struct {
	query Query
}

func (h GetAllegianceHandler) GetAllegiance(
	ctx context.Context,
	_ *connect.Request[planetv1.GetAllegianceRequest],
) (*connect.Response[planetv1.GetAllegianceResponse], error) {
	allegiance, err := h.query.Allegiance(ctx, allegiances.AccountOf(clicks.PayerOf(ctx)))
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(allegiance), nil
}
