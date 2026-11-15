package retell_missions_paid_handler

import (
	"context"

	"connectrpc.com/connect"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/retell_missions_paid_usecase"
)

func New(useCase retell_missions_paid_usecase.Executor) RetellMissionsPaidHandler {
	return RetellMissionsPaidHandler{useCase: useCase}
}

type RetellMissionsPaidHandler struct {
	useCase retell_missions_paid_usecase.Executor
}

func (h RetellMissionsPaidHandler) RetellMissionsPaid(
	ctx context.Context,
	req *connect.Request[missionsv1.RetellMissionsPaidRequest],
) (*connect.Response[missionsv1.RetellMissionsPaidResponse], error) {
	out, err := h.useCase.Execute(ctx, retell_missions_paid_usecase.In{DryRun: req.Msg.GetDryRun()})
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&missionsv1.RetellMissionsPaidResponse{
		Accounts: uint32(out.Accounts), //nolint:gosec // counted from the accounts paid, never negative or past 2^32.
	}), nil
}
