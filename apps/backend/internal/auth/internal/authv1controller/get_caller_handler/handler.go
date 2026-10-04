package get_caller_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type UseCase interface {
	Execute(ctx context.Context, cookieHeader string) (accounts.AccountID, error)
}

func New(useCase UseCase) GetCallerHandler {
	return GetCallerHandler{useCase: useCase}
}

type GetCallerHandler struct {
	useCase UseCase
}

func (h GetCallerHandler) GetCaller(
	ctx context.Context,
	req *connect.Request[authv1.GetCallerRequest],
) (*connect.Response[authv1.GetCallerResponse], error) {
	account, err := h.useCase.Execute(ctx, req.Msg.GetCookie())
	if errors.Is(err, accounts.ErrNoAccount) {
		return connect.NewResponse(&authv1.GetCallerResponse{}), nil
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	return connect.NewResponse(&authv1.GetCallerResponse{AccountId: account.String()}), nil
}
