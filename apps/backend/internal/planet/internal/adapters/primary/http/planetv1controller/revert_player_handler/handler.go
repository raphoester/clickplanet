package revert_player_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/revert_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in revert_player_usecase.In) (revert_player_usecase.Out, error)
}

func New(useCase UseCase) RevertPlayerHandler {
	return RevertPlayerHandler{useCase: useCase}
}

type RevertPlayerHandler struct {
	useCase UseCase
}

func (h RevertPlayerHandler) RevertPlayer(
	ctx context.Context,
	req *connect.Request[planetv1.RevertPlayerRequest],
) (*connect.Response[planetv1.RevertPlayerResponse], error) {
	out, err := h.useCase.Execute(ctx, revert_player_usecase.In{
		Scope:  req.Msg.GetScope(),
		DryRun: req.Msg.GetDryRun(),
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.RevertPlayerResponse{
			Scope:    out.Scope,
			Touched:  uint32(out.Touched),  //nolint:gosec // a tile count, bounded by the map.
			Held:     uint32(out.Held),     //nolint:gosec // a tile count, bounded by the map.
			Restored: uint32(out.Restored), //nolint:gosec // a tile count, bounded by the map.
		}), nil
	case errors.Is(err, clicks.ErrInvalidScope):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
