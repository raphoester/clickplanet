// Package resolve_account_handler serves auth.v1.InternalService/ResolveAccount, for other modules.
package resolve_account_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resolve_account_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in resolve_account_usecase.In) (*resolve_account_usecase.Out, error)
}

func New(useCase UseCase) ResolveAccountHandler {
	return ResolveAccountHandler{useCase: useCase}
}

type ResolveAccountHandler struct {
	useCase UseCase
}

// ResolveAccount answers an empty account id for no account: on the wire, absence is not an error.
func (h ResolveAccountHandler) ResolveAccount(
	ctx context.Context,
	req *connect.Request[authv1.ResolveAccountRequest],
) (*connect.Response[authv1.ResolveAccountResponse], error) {
	out, err := h.useCase.Execute(ctx, resolve_account_usecase.In{
		CookieHeader: req.Msg.GetCookieHeader(),
		Create:       req.Msg.GetCreate(),
	})
	if errors.Is(err, accounts.ErrNoAccount) {
		return connect.NewResponse(&authv1.ResolveAccountResponse{}), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to resolve the account: %w", err)
	}

	return connect.NewResponse(&authv1.ResolveAccountResponse{
		AccountId: out.Account.String(),
		SetCookie: out.SetCookie,
	}), nil
}
