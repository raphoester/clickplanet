package rebuild_stats_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/usecases/rebuild_stats_usecase"
)

func New(useCase rebuild_stats_usecase.Executor) RebuildStatsHandler {
	return RebuildStatsHandler{useCase: useCase}
}

type RebuildStatsHandler struct {
	useCase rebuild_stats_usecase.Executor
}

func (h RebuildStatsHandler) RebuildStats(
	ctx context.Context,
	_ *connect.Request[playerv1.RebuildStatsRequest],
) (*connect.Response[playerv1.RebuildStatsResponse], error) {
	out, err := h.useCase.Execute(ctx)
	if errors.Is(err, takes.ErrNotStarted) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&playerv1.RebuildStatsResponse{FromPosition: uint64(out.From)}), nil
}
