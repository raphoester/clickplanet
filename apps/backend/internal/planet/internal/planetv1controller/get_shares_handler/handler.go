package get_shares_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_shares_usecase"
)

type UseCase interface {
	Execute(ctx context.Context) get_shares_usecase.Out
}

func New(useCase UseCase) GetSharesHandler {
	return GetSharesHandler{useCase: useCase}
}

type GetSharesHandler struct {
	useCase UseCase
}

func (h GetSharesHandler) GetShares(
	ctx context.Context,
	_ *connect.Request[planetv1.GetSharesRequest],
) (*connect.Response[planetv1.GetSharesResponse], error) {
	out := h.useCase.Execute(ctx)

	countries := make([]*planetv1.CountryTiles, len(out.Holdings))
	for i, holding := range out.Holdings {
		countries[i] = &planetv1.CountryTiles{Country: holding.Country, Tiles: holding.Tiles}
	}

	return connect.NewResponse(&planetv1.GetSharesResponse{MapTiles: out.MapTiles, Countries: countries}), nil
}
