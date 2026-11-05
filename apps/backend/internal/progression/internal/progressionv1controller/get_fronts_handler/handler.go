package get_fronts_handler

import (
	"context"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/caller"
)

type Query interface {
	Fronts(ctx context.Context, account accounts.AccountID) (*progressionv1.GetFrontsResponse, error)
}

func New(query Query) GetFrontsHandler {
	return GetFrontsHandler{query: query}
}

type GetFrontsHandler struct {
	query Query
}

func (h GetFrontsHandler) GetFronts(
	ctx context.Context,
	_ *connect.Request[progressionv1.GetFrontsRequest],
) (*connect.Response[progressionv1.GetFrontsResponse], error) {
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
