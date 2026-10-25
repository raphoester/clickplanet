package collect_mission_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions/usecases/collect_mission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in collect_mission_usecase.In) (collect_mission_usecase.Out, error)
}

func New(useCase UseCase) CollectMissionHandler {
	return CollectMissionHandler{useCase: useCase}
}

type CollectMissionHandler struct {
	useCase UseCase
}

func (h CollectMissionHandler) CollectMission(
	ctx context.Context,
	req *connect.Request[playerv1.CollectMissionRequest],
) (*connect.Response[playerv1.CollectMissionResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	out, err := h.useCase.Execute(ctx, collect_mission_usecase.In{
		Account: account, Linked: cpctx.GetLinked(ctx), Slot: req.Msg.GetSlot(),
	})
	switch {
	case errors.Is(err, collect_mission_usecase.ErrNotLinked):
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, missions.ErrNoSlot):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, missions.ErrNotDone), errors.Is(err, missions.ErrCollected),
		errors.Is(err, collect_mission_usecase.ErrNoRoom):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&playerv1.CollectMissionResponse{
		Kind: playermessage.RewardKind(out.Reward.Kind), Amount: out.Reward.Amount,
	}), nil
}
