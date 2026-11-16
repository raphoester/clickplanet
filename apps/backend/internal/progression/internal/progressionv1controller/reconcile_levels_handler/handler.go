package reconcile_levels_handler

import (
	"context"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/usecases/reconcile_levels_usecase"
)

func New(useCase reconcile_levels_usecase.Executor) ReconcileLevelsHandler {
	return ReconcileLevelsHandler{useCase: useCase}
}

type ReconcileLevelsHandler struct {
	useCase reconcile_levels_usecase.Executor
}

func (h ReconcileLevelsHandler) ReconcileLevels(
	ctx context.Context,
	req *connect.Request[progressionv1.ReconcileLevelsRequest],
) (*connect.Response[progressionv1.ReconcileLevelsResponse], error) {
	reconciled, err := h.useCase.Execute(ctx, req.Msg.GetDryRun())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&progressionv1.ReconcileLevelsResponse{
		Raised:  uint32(reconciled.Raised),  //nolint:gosec // counted from the stats' rows, never negative or past 2^32.
		Lowered: uint32(reconciled.Lowered), //nolint:gosec // as above.
		Removed: uint32(reconciled.Removed), //nolint:gosec // as above.
	}), nil
}
