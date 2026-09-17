// Package sign_out_handler serves auth.v1.AuthService/SignOut.
package sign_out_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
)

type UseCase interface {
	Execute(ctx context.Context, cookieHeader string) (string, error)
}

func New(useCase UseCase) SignOutHandler {
	return SignOutHandler{useCase: useCase}
}

type SignOutHandler struct {
	useCase UseCase
}

func (h SignOutHandler) SignOut(
	ctx context.Context,
	req *connect.Request[authv1.SignOutRequest],
) (*connect.Response[authv1.SignOutResponse], error) {
	setCookie, err := h.useCase.Execute(ctx, req.Header().Get("Cookie"))
	if err != nil {
		return nil, fmt.Errorf("failed to sign out: %w", err)
	}

	res := connect.NewResponse(&authv1.SignOutResponse{})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", setCookie)
	return res, nil
}
