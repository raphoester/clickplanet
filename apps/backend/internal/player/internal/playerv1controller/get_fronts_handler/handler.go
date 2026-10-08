package get_fronts_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
)

type Query interface {
	Fronts(ctx context.Context, account players.AccountID) (*playerv1.GetFrontsResponse, error)
}

func New(query Query) GetFrontsHandler {
	return GetFrontsHandler{query: query}
}

type GetFrontsHandler struct {
	query Query
}

func (h GetFrontsHandler) GetFronts(
	ctx context.Context,
	_ *connect.Request[playerv1.GetFrontsRequest],
) (*connect.Response[playerv1.GetFrontsResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}
	fronts, err := h.query.Fronts(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(fronts), nil
}
