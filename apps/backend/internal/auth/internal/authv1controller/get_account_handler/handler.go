// Package get_account_handler serves auth.v1.InternalService/GetAccount.
package get_account_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type UseCase interface {
	Execute(ctx context.Context, account accounts.AccountID) (*accounts.Account, error)
}

func New(useCase UseCase) GetAccountHandler {
	return GetAccountHandler{useCase: useCase}
}

type GetAccountHandler struct {
	useCase UseCase
}

// GetAccount answers linked false for an id that is not an account, as for an account that does not exist:
// the caller asks whether it may trust the account, and neither may be trusted.
func (h GetAccountHandler) GetAccount(
	ctx context.Context,
	req *connect.Request[authv1.GetAccountRequest],
) (*connect.Response[authv1.GetAccountResponse], error) {
	id, err := accounts.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return connect.NewResponse(&authv1.GetAccountResponse{}), nil
	}

	account, err := h.useCase.Execute(ctx, id)
	if errors.Is(err, accounts.ErrAccountNotFound) {
		return connect.NewResponse(&authv1.GetAccountResponse{}), nil
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	return connect.NewResponse(&authv1.GetAccountResponse{Linked: account.Linked()}), nil
}
