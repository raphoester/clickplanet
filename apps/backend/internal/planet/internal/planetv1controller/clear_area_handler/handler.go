package clear_area_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/clear_area_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in clear_area_usecase.In) (clear_area_usecase.Out, error)
}

func New(useCase UseCase) ClearAreaHandler {
	return ClearAreaHandler{useCase: useCase}
}

type ClearAreaHandler struct {
	useCase UseCase
}

func (h ClearAreaHandler) ClearArea(
	ctx context.Context,
	req *connect.Request[planetv1.ClearAreaRequest],
) (*connect.Response[planetv1.ClearAreaResponse], error) {
	out, err := h.useCase.Execute(ctx, clear_area_usecase.In{
		Tile: req.Msg.GetTileId(), Rings: req.Msg.GetRings(), DryRun: req.Msg.GetDryRun(),
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.ClearAreaResponse{
			Tiles:   uint32(out.Tiles),   //nolint:gosec // a tile count, bounded by the map.
			Owned:   uint32(out.Owned),   //nolint:gosec // a tile count, bounded by the map.
			Cleared: uint32(out.Cleared), //nolint:gosec // a tile count, bounded by the map.
		}), nil
	case errors.Is(err, clicks.ErrTileOutOfRange), errors.Is(err, clicks.ErrInvalidArea):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
