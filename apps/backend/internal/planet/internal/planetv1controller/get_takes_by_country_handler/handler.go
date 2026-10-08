package get_takes_by_country_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Query interface {
	TakesByCountry(ctx context.Context, account ledger.AccountID) (*planetv1.GetTakesByCountryResponse, error)
}

func New(query Query) GetTakesByCountryHandler {
	return GetTakesByCountryHandler{query: query}
}

type GetTakesByCountryHandler struct {
	query Query
}

func (h GetTakesByCountryHandler) GetTakesByCountry(
	ctx context.Context,
	req *connect.Request[planetv1.GetTakesByCountryRequest],
) (*connect.Response[planetv1.GetTakesByCountryResponse], error) {
	account, err := ledger.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	takes, err := h.query.TakesByCountry(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(takes), nil
}
