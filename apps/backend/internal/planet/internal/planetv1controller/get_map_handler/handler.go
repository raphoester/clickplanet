package get_map_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in get_map_usecase.In) (clicks.DenseBatch, error)
}

func New(useCase UseCase) GetMapHandler {
	return GetMapHandler{useCase: useCase}
}

type GetMapHandler struct {
	useCase UseCase
}

func (h GetMapHandler) GetMap(
	ctx context.Context,
	req *connect.Request[planetv1.GetMapRequest],
) (*connect.Response[planetv1.GetMapResponse], error) {
	batch, err := h.useCase.Execute(ctx, get_map_usecase.In{
		Start: req.Msg.GetStartTileId(),
		End:   req.Msg.GetEndTileId(),
	})
	if err != nil {
		return nil, toConnect(err)
	}

	defenders := make([]*planetv1.TileDefenders, len(batch.Defenders))
	for i, tile := range batch.Defenders {
		defenders[i] = &planetv1.TileDefenders{
			TileId:    tile.Tile,
			Defenders: uint32(tile.Defenders), //nolint:gosec // a byte in the tile storage.
		}
	}

	return connect.NewResponse(&planetv1.GetMapResponse{
		StartTileId: batch.Start,
		Codes:       batch.Codes,
		Tiles:       batch.Tiles,
		Defenders:   defenders,
	}), nil
}
