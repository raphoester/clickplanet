package get_allegiance_of_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
)

type Query interface {
	Allegiance(ctx context.Context, account allegiances.Account) (*planetv1.GetAllegianceResponse, error)
}

func New(query Query) GetAllegianceOfHandler {
	return GetAllegianceOfHandler{query: query}
}

type GetAllegianceOfHandler struct {
	query Query
}

func (h GetAllegianceOfHandler) GetAllegianceOf(
	ctx context.Context,
	req *connect.Request[planetv1.GetAllegianceOfRequest],
) (*connect.Response[planetv1.GetAllegianceOfResponse], error) {
	account, err := allegiances.AccountFrom(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	held, err := h.query.Allegiance(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.GetAllegianceOfResponse{Allegiance: held.GetAllegiance()}), nil
}
