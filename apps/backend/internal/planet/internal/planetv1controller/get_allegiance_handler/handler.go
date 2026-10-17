package get_allegiance_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Query interface {
	Allegiance(account allegiances.Account) *planetv1.GetAllegianceResponse
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
	return connect.NewResponse(h.query.Allegiance(allegiances.AccountOf(clicks.PayerOf(ctx)))), nil
}
