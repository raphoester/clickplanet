package retell_missions_done_handler

import (
	"context"

	"connectrpc.com/connect"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/retell_missions_done_usecase"
)

func New(useCase retell_missions_done_usecase.Executor) RetellMissionsDoneHandler {
	return RetellMissionsDoneHandler{useCase: useCase}
}

type RetellMissionsDoneHandler struct {
	useCase retell_missions_done_usecase.Executor
}

func (h RetellMissionsDoneHandler) RetellMissionsDone(
	ctx context.Context,
	req *connect.Request[missionsv1.RetellMissionsDoneRequest],
) (*connect.Response[missionsv1.RetellMissionsDoneResponse], error) {
	out, err := h.useCase.Execute(ctx, retell_missions_done_usecase.In{DryRun: req.Msg.GetDryRun()})
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&missionsv1.RetellMissionsDoneResponse{
		Accounts: uint32(out.Accounts), //nolint:gosec // counted from the accounts with a mission done, never negative or past 2^32.
	}), nil
}
