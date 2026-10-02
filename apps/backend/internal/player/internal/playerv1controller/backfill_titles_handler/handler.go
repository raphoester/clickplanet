package backfill_titles_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/backfill_titles_usecase"
)

func New(useCase backfill_titles_usecase.Executor) BackfillTitlesHandler {
	return BackfillTitlesHandler{useCase: useCase}
}

type BackfillTitlesHandler struct {
	useCase backfill_titles_usecase.Executor
}

func (h BackfillTitlesHandler) BackfillTitles(
	ctx context.Context,
	_ *connect.Request[playerv1.BackfillTitlesRequest],
) (*connect.Response[playerv1.BackfillTitlesResponse], error) {
	backfill, err := h.useCase.Execute(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&playerv1.BackfillTitlesResponse{
		Accounts: uint32(backfill.Accounts), //nolint:gosec // counted from accounts, never negative or past 2^32.
	}), nil
}
