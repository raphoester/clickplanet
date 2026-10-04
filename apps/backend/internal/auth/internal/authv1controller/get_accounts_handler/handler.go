package get_accounts_handler

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type Query interface {
	Accounts(ctx context.Context, accounts []accounts.AccountID) (*authv1.GetAccountsResponse, error)
}

func New(query Query) GetAccountsHandler {
	return GetAccountsHandler{query: query}
}

type GetAccountsHandler struct {
	query Query
}

func (h GetAccountsHandler) GetAccounts(
	ctx context.Context,
	req *connect.Request[authv1.GetAccountsRequest],
) (*connect.Response[authv1.GetAccountsResponse], error) {
	asked := make([]accounts.AccountID, 0, len(req.Msg.GetAccountIds()))
	for _, value := range req.Msg.GetAccountIds() {
		account, err := accounts.AccountIDOf(value)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		asked = append(asked, account)
	}

	found, err := h.query.Accounts(ctx, asked)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(found), nil
}
