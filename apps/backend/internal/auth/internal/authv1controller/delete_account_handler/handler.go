// Package delete_account_handler serves auth.v1.AuthService/DeleteAccount.
package delete_account_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/delete_account_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, cookieHeader string) (*delete_account_usecase.Out, error)
}

func New(useCase UseCase) DeleteAccountHandler {
	return DeleteAccountHandler{useCase: useCase}
}

type DeleteAccountHandler struct {
	useCase UseCase
}

func (h DeleteAccountHandler) DeleteAccount(
	ctx context.Context,
	req *connect.Request[authv1.DeleteAccountRequest],
) (*connect.Response[authv1.DeleteAccountResponse], error) {
	out, err := h.useCase.Execute(ctx, req.Header().Get("Cookie"))
	if errors.Is(err, accounts.ErrNoAccount) {
		return nil, connect.NewError(connect.CodeUnauthenticated, accounts.ErrNoAccount)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to delete the account: %w", err)
	}

	res := connect.NewResponse(&authv1.DeleteAccountResponse{})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", out.SetCookie)
	return res, nil
}
