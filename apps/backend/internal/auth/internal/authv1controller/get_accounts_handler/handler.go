package get_accounts_handler

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type UseCase interface {
	Execute(ctx context.Context, accounts []accounts.AccountID) ([]*accounts.Account, error)
}

func New(useCase UseCase) GetAccountsHandler {
	return GetAccountsHandler{useCase: useCase}
}

type GetAccountsHandler struct {
	useCase UseCase
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

	found, err := h.useCase.Execute(ctx, asked)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := &authv1.GetAccountsResponse{Accounts: make([]*authv1.Account, 0, len(found))}
	for _, account := range found {
		res.Accounts = append(res.Accounts, &authv1.Account{
			AccountId: account.ID.String(), Linked: account.Linked(), CreatedAtUnixMs: account.CreatedAt.UnixMilli(),
		})
	}
	return connect.NewResponse(res), nil
}
