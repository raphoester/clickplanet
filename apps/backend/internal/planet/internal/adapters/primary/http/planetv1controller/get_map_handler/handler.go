// Package get_map_handler serves planet.v1.ClickService/GetMap.
package get_map_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map"
)

type UseCase interface {
	Execute(ctx context.Context, in get_map.In) (clicks.DenseBatch, error)
}

func New(useCase UseCase) GetMapHandler {
	return GetMapHandler{useCase: useCase}
}

type GetMapHandler struct {
	useCase UseCase
}

// The batch is copied out field by field rather than converted: the proto
// message and the domain struct have different reasons to change.
func (h GetMapHandler) GetMap(
	ctx context.Context,
	req *connect.Request[planetv1.GetMapRequest],
) (*connect.Response[planetv1.GetMapResponse], error) {
	batch, err := h.useCase.Execute(ctx, get_map.In{
		Start: req.Msg.GetStartTileId(),
		End:   req.Msg.GetEndTileId(),
	})
	if err != nil {
		return nil, toConnect(err)
	}

	return connect.NewResponse(&planetv1.GetMapResponse{
		StartTileId: batch.Start,
		Codes:       batch.Codes,
		Tiles:       batch.Tiles,
	}), nil
}
