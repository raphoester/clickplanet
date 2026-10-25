package recount_stats_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/recount_stats_usecase"
)

func New(useCase recount_stats_usecase.Executor) RecountStatsHandler {
	return RecountStatsHandler{useCase: useCase}
}

type RecountStatsHandler struct {
	useCase recount_stats_usecase.Executor
}

func (h RecountStatsHandler) RecountStats(
	ctx context.Context,
	_ *connect.Request[playerv1.RecountStatsRequest],
) (*connect.Response[playerv1.RecountStatsResponse], error) {
	recounted, err := h.useCase.Execute(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&playerv1.RecountStatsResponse{
		Recounted: uint32(recounted), //nolint:gosec // counted from the accounts with stats, never negative or past 2^32.
	}), nil
}
