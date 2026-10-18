package admin_sign_out_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
)

type UseCase interface {
	Execute(ctx context.Context, cookieHeader string) error
}

func New(useCase UseCase) AdminSignOutHandler {
	return AdminSignOutHandler{useCase: useCase}
}

type AdminSignOutHandler struct {
	useCase UseCase
}

func (h AdminSignOutHandler) SignOut(
	ctx context.Context,
	req *connect.Request[authv1.AdminSignOutRequest],
) (*connect.Response[authv1.AdminSignOutResponse], error) {
	if err := h.useCase.Execute(ctx, req.Header().Get("Cookie")); err != nil {
		return nil, fmt.Errorf("failed to sign out of the admin page: %w", err)
	}

	res := connect.NewResponse(&authv1.AdminSignOutResponse{})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", adminsignin.ExpiredCookie())
	return res, nil
}
