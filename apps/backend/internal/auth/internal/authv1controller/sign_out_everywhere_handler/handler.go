// Package sign_out_everywhere_handler serves auth.v1.AuthService/SignOutEverywhere.
package sign_out_everywhere_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type UseCase interface {
	Execute(ctx context.Context, cookieHeader string) (string, error)
}

func New(useCase UseCase) SignOutEverywhereHandler {
	return SignOutEverywhereHandler{useCase: useCase}
}

type SignOutEverywhereHandler struct {
	useCase UseCase
}

func (h SignOutEverywhereHandler) SignOutEverywhere(
	ctx context.Context,
	req *connect.Request[authv1.SignOutEverywhereRequest],
) (*connect.Response[authv1.SignOutEverywhereResponse], error) {
	setCookie, err := h.useCase.Execute(ctx, req.Header().Get("Cookie"))
	if errors.Is(err, accounts.ErrNoAccount) {
		return nil, connect.NewError(connect.CodeUnauthenticated, accounts.ErrNoAccount)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to sign out everywhere: %w", err)
	}

	res := connect.NewResponse(&authv1.SignOutEverywhereResponse{})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", setCookie)
	return res, nil
}
