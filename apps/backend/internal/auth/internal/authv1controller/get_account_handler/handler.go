package get_account_handler

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type Query interface {
	Account(ctx context.Context, account accounts.AccountID) (*authv1.GetAccountResponse, error)
}

func New(query Query) GetAccountHandler {
	return GetAccountHandler{query: query}
}

type GetAccountHandler struct {
	query Query
}

func (h GetAccountHandler) GetAccount(
	ctx context.Context,
	req *connect.Request[authv1.GetAccountRequest],
) (*connect.Response[authv1.GetAccountResponse], error) {
	id, err := accounts.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return connect.NewResponse(&authv1.GetAccountResponse{}), nil
	}

	account, err := h.query.Account(ctx, id)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(account), nil
}
