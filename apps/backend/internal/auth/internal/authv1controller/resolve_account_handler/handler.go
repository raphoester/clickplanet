// Package resolve_account_handler serves auth.v1.InternalService/ResolveAccount, for other modules.
package resolve_account_handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resolve_account_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in resolve_account_usecase.In) (resolve_account_usecase.Out, error)
}

func New(useCase UseCase) ResolveAccountHandler {
	return ResolveAccountHandler{useCase: useCase}
}

type ResolveAccountHandler struct {
	useCase UseCase
}

func (h ResolveAccountHandler) ResolveAccount(
	ctx context.Context,
	req *connect.Request[authv1.ResolveAccountRequest],
) (*connect.Response[authv1.ResolveAccountResponse], error) {
	out, err := h.useCase.Execute(ctx, resolve_account_usecase.In{
		CookieHeader: req.Msg.GetCookieHeader(),
		Create:       req.Msg.GetCreate(),
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it as internal.
	}

	response := &authv1.ResolveAccountResponse{SetCookie: out.SetCookie}
	if out.Account != uuid.Nil {
		response.AccountId = out.Account.String()
	}

	return connect.NewResponse(response), nil
}
