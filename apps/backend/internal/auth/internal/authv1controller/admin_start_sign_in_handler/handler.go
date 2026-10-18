package admin_start_sign_in_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin/usecases/start_admin_sign_in_usecase"
)

type UseCase interface {
	Execute(ctx context.Context) (*start_admin_sign_in_usecase.Out, error)
}

func New(useCase UseCase) AdminStartSignInHandler {
	return AdminStartSignInHandler{useCase: useCase}
}

type AdminStartSignInHandler struct {
	useCase UseCase
}

func (h AdminStartSignInHandler) StartSignIn(
	ctx context.Context,
	_ *connect.Request[authv1.AdminStartSignInRequest],
) (*connect.Response[authv1.AdminStartSignInResponse], error) {
	out, err := h.useCase.Execute(ctx)
	if errors.Is(err, adminsignin.ErrSignInOff) {
		return nil, connect.NewError(connect.CodeUnimplemented, adminsignin.ErrSignInOff)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to start the admin sign-in: %w", err)
	}

	res := connect.NewResponse(&authv1.AdminStartSignInResponse{AuthorizationUrl: out.AuthorizationURL})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", out.SetCookie)
	return res, nil
}
