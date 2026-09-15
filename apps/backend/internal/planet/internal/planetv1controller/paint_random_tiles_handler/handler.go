package paint_random_tiles_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in paint_random_tiles_usecase.In) (paint_random_tiles_usecase.Out, error)
}

func New(useCase UseCase) PaintRandomTilesHandler {
	return PaintRandomTilesHandler{useCase: useCase}
}

type PaintRandomTilesHandler struct {
	useCase UseCase
}

func (h PaintRandomTilesHandler) PaintRandomTiles(
	ctx context.Context,
	req *connect.Request[planetv1.PaintRandomTilesRequest],
) (*connect.Response[planetv1.PaintRandomTilesResponse], error) {
	out, err := h.useCase.Execute(ctx, paint_random_tiles_usecase.In{
		Flag:      req.Msg.GetFlagCountryId(),
		Area:      req.Msg.GetAreaCountryId(),
		Count:     int(req.Msg.GetCount()),
		Proximity: req.Msg.GetProximity(),
		DryRun:    req.Msg.GetDryRun(),
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.PaintRandomTilesResponse{
			Eligible:    uint32(out.Eligible),    //nolint:gosec // a tile count, bounded by the map.
			Picked:      uint32(out.Picked),      //nolint:gosec // a tile count, bounded by the map.
			Painted:     uint32(out.Painted),     //nolint:gosec // a tile count, bounded by the map.
			OutsideArea: uint32(out.OutsideArea), //nolint:gosec // a tile count, bounded by the map.
		}), nil
	case errors.Is(err, clicks.ErrUnknownCountry),
		errors.Is(err, paint_random_tiles_usecase.ErrInvalidCount),
		errors.Is(err, paint_random_tiles_usecase.ErrInvalidProximity):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
