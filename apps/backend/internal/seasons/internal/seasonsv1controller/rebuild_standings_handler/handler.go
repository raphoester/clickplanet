package rebuild_standings_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/rebuild_standings_usecase"
)

func New(useCase rebuild_standings_usecase.Executor) RebuildStandingsHandler {
	return RebuildStandingsHandler{useCase: useCase}
}

type RebuildStandingsHandler struct {
	useCase rebuild_standings_usecase.Executor
}

func (h RebuildStandingsHandler) RebuildStandings(
	ctx context.Context,
	_ *connect.Request[seasonsv1.RebuildStandingsRequest],
) (*connect.Response[seasonsv1.RebuildStandingsResponse], error) {
	out, err := h.useCase.Execute(ctx)
	if errors.Is(err, standings.ErrNotStarted) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&seasonsv1.RebuildStandingsResponse{FromPosition: uint64(out.From)}), nil
}
