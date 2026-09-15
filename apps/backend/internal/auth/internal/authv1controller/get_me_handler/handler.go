// Package get_me_handler serves auth.v1.AuthService/GetMe.
package get_me_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resolve_account_usecase"
)

var ErrNoAccount = errors.New("this browser has no account")

type UseCase interface {
	Execute(ctx context.Context, in resolve_account_usecase.In) (resolve_account_usecase.Out, error)
}

func New(useCase UseCase) GetMeHandler {
	return GetMeHandler{useCase: useCase}
}

type GetMeHandler struct {
	useCase UseCase
}

// GetMe creates nothing: only a mint, after Turnstile, gives a browser an account.
func (h GetMeHandler) GetMe(
	ctx context.Context,
	req *connect.Request[authv1.GetMeRequest],
) (*connect.Response[authv1.GetMeResponse], error) {
	out, err := h.useCase.Execute(ctx, resolve_account_usecase.In{CookieHeader: req.Header().Get("Cookie")})
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it as internal.
	}

	if out.Account == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoAccount)
	}

	res := connect.NewResponse(&authv1.GetMeResponse{AccountId: out.Account.String()})
	res.Header().Set("Cache-Control", "no-store")
	if out.SetCookie != "" {
		res.Header().Add("Set-Cookie", out.SetCookie)
	}

	return res, nil
}
