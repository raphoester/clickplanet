package restore_area_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/restore_area_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/adminplayer"
)

type UseCase interface {
	Execute(ctx context.Context, in restore_area_usecase.In) (restore_area_usecase.Out, error)
}

func New(useCase UseCase) RestoreAreaHandler {
	return RestoreAreaHandler{useCase: useCase}
}

type RestoreAreaHandler struct {
	useCase UseCase
}

func (h RestoreAreaHandler) RestoreArea(
	ctx context.Context,
	req *connect.Request[planetv1.RestoreAreaRequest],
) (*connect.Response[planetv1.RestoreAreaResponse], error) {
	in := restore_area_usecase.In{Tile: req.Msg.GetTileId(), Rings: req.Msg.GetRings(), DryRun: req.Msg.GetDryRun()}
	if before := req.Msg.GetBefore(); before != nil {
		in.Before = before.AsTime()
	}

	out, err := h.useCase.Execute(ctx, in)

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.RestoreAreaResponse{
			Tiles:         uint32(out.Tiles),         //nolint:gosec // a tile count, bounded by the map.
			Changed:       uint32(out.Changed),       //nolint:gosec // a tile count, bounded by the map.
			Restored:      uint32(out.Restored),      //nolint:gosec // a tile count, bounded by the map.
			PaintersTotal: uint32(out.PaintersTotal), //nolint:gosec // a player count, bounded by the ledger.
			Painters:      adminplayer.Encode(out.Painters),
			KeptSince:     timestamppb.New(out.KeptSince),
		}), nil
	case errors.Is(err, clicks.ErrTileOutOfRange), errors.Is(err, clicks.ErrInvalidArea),
		errors.Is(err, restore_area_usecase.ErrNoTime):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, restore_area_usecase.ErrPastTheLedger):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return nil, err
	}
}
