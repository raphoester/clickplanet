// Package start_sign_in_handler serves auth.v1.AuthService/StartSignIn.
package start_sign_in_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/authprovider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_sign_in_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, provider string) (*start_sign_in_usecase.Out, error)
}

func New(useCase UseCase) StartSignInHandler {
	return StartSignInHandler{useCase: useCase}
}

type StartSignInHandler struct {
	useCase UseCase
}

// StartSignIn answers Unimplemented, which is HTTP 404, while sign-in is off: the client hides the button on it.
func (h StartSignInHandler) StartSignIn(
	ctx context.Context,
	req *connect.Request[authv1.StartSignInRequest],
) (*connect.Response[authv1.StartSignInResponse], error) {
	out, err := h.useCase.Execute(ctx, authprovider.Decode(req.Msg.GetProvider()))
	switch {
	case errors.Is(err, signin.ErrSignInOff):
		return nil, connect.NewError(connect.CodeUnimplemented, signin.ErrSignInOff)
	case errors.Is(err, signin.ErrUnknownProvider):
		return nil, connect.NewError(connect.CodeInvalidArgument, signin.ErrUnknownProvider)
	case err != nil:
		return nil, fmt.Errorf("failed to start the sign-in: %w", err)
	}

	res := connect.NewResponse(&authv1.StartSignInResponse{AuthorizationUrl: out.AuthorizationURL})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", out.SetCookie)
	return res, nil
}
