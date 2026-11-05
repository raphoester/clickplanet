package get_appearances_handler

import (
	"context"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/caller"
)

type Query interface {
	Appearances(ctx context.Context, account casts.AccountID) (*clipsv1.GetAppearancesResponse, error)
}

func New(query Query) GetAppearancesHandler {
	return GetAppearancesHandler{query: query}
}

type GetAppearancesHandler struct {
	query Query
}

func (h GetAppearancesHandler) GetAppearances(
	ctx context.Context,
	_ *connect.Request[clipsv1.GetAppearancesRequest],
) (*connect.Response[clipsv1.GetAppearancesResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already a connect error.
	}

	answer, err := h.query.Appearances(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(answer), nil
}
