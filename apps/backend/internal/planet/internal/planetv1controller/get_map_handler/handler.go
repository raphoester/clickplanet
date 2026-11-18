package get_map_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

type UseCase interface {
	Execute(ctx context.Context, in get_map_usecase.In) (clicks.DenseBatch, error)
}

// Boards names a board by its row.
type Boards interface {
	Name(id clicks.BoardID) clicks.BoardName
}

func New(useCase UseCase, boards Boards) GetMapHandler {
	return GetMapHandler{useCase: useCase, boards: boards}
}

type GetMapHandler struct {
	useCase UseCase
	boards  Boards
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

	answer := planetmessage.Map(batch)
	answer.Board = string(h.boards.Name(batch.Board))
	return connect.NewResponse(answer), nil
}
