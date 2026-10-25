package collect_mission_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/collect_mission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/missionmessage"
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
	req *connect.Request[missionsv1.CollectMissionRequest],
) (*connect.Response[missionsv1.CollectMissionResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	out, err := h.useCase.Execute(ctx, collect_mission_usecase.In{
		Account: account, Linked: cpctx.GetLinked(ctx), Mission: req.Msg.GetMissionId(),
	})
	switch {
	case errors.Is(err, collect_mission_usecase.ErrNotLinked):
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, missions.ErrNoMission):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, missions.ErrNotDone), errors.Is(err, missions.ErrCollected),
		errors.Is(err, collect_mission_usecase.ErrNoRoom):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&missionsv1.CollectMissionResponse{
		Kind: missionmessage.RewardKind(out.Reward.Kind), Amount: out.Reward.Amount,
	}), nil
}
