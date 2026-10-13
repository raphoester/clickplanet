package find_account_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/find_account_handler/account_query"
)

type Query interface {
	Account(ctx context.Context, name string) (*playerv1.FindAccountResponse, error)
}

func New(query Query) FindAccountHandler {
	return FindAccountHandler{query: query}
}

type FindAccountHandler struct {
	query Query
}

func (h FindAccountHandler) FindAccount(
	ctx context.Context,
	req *connect.Request[playerv1.FindAccountRequest],
) (*connect.Response[playerv1.FindAccountResponse], error) {
	account, err := h.query.Account(ctx, req.Msg.GetName())
	switch {
	case errors.Is(err, account_query.ErrNoAccount):
		return nil, connect.NewError(connect.CodeNotFound, account_query.ErrNoAccount)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	res := connect.NewResponse(account)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
