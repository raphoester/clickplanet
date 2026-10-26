package get_missions_handler

import (
	"context"

	"connectrpc.com/connect"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/missionmessage"
)

type UseCase interface {
	Execute(ctx context.Context, in get_missions_usecase.In) (get_missions_usecase.Out, error)
}

func New(useCase UseCase) GetMissionsHandler {
	return GetMissionsHandler{useCase: useCase}
}

type GetMissionsHandler struct {
	useCase UseCase
}

func (h GetMissionsHandler) GetMissions(
	ctx context.Context,
	_ *connect.Request[missionsv1.GetMissionsRequest],
) (*connect.Response[missionsv1.GetMissionsResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	out, err := h.useCase.Execute(ctx, get_missions_usecase.In{Account: account})
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	answer := &missionsv1.GetMissionsResponse{
		CountryId: out.Flag, Enlisted: out.Sheet.Enlisted(), EndsAtUnixMs: out.Day.EndsAt().UnixMilli(),
	}
	for _, status := range out.Sheet.Statuses(out.Brief) {
		mission, err := missionmessage.Mission(status)
		if err != nil {
			return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
		}
		answer.Missions = append(answer.Missions, mission)
	}
	return connect.NewResponse(answer), nil
}
