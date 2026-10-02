package reconcile_titles_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/reconcile_titles_usecase"
)

func New(useCase reconcile_titles_usecase.Executor) ReconcileTitlesHandler {
	return ReconcileTitlesHandler{useCase: useCase}
}

type ReconcileTitlesHandler struct {
	useCase reconcile_titles_usecase.Executor
}

func (h ReconcileTitlesHandler) ReconcileTitles(
	ctx context.Context,
	_ *connect.Request[playerv1.ReconcileTitlesRequest],
) (*connect.Response[playerv1.ReconcileTitlesResponse], error) {
	reconciled, err := h.useCase.Execute(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&playerv1.ReconcileTitlesResponse{
		Granted: uint32(reconciled.Granted), //nolint:gosec // counted from titles held, never negative or past 2^32.
		Revoked: uint32(reconciled.Revoked), //nolint:gosec // as above.
	}), nil
}
