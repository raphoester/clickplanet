// Package start_sign_in_handler serves auth.v1.AuthService/StartSignIn.
package start_sign_in_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/authprovider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_sign_in_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in start_sign_in_usecase.In) (*start_sign_in_usecase.Out, error)
}

func New(useCase UseCase) StartSignInHandler {
	return StartSignInHandler{useCase: useCase}
}

type StartSignInHandler struct {
	useCase UseCase
}

// StartSignIn answers Unimplemented, which is HTTP 404, while sign-in is off. A client learns that first from GetSignInOptions.
// A link from a browser with no account is Unauthenticated: there is nothing to link to.
func (h StartSignInHandler) StartSignIn(
	ctx context.Context,
	req *connect.Request[authv1.StartSignInRequest],
) (*connect.Response[authv1.StartSignInResponse], error) {
	out, err := h.useCase.Execute(ctx, start_sign_in_usecase.In{
		Provider:     authprovider.NameOf(req.Msg.GetProvider()),
		Intent:       intentOf(req.Msg.GetIntent()),
		CookieHeader: req.Header().Get("Cookie"),
	})
	switch {
	case errors.Is(err, signin.ErrSignInOff):
		return nil, connect.NewError(connect.CodeUnimplemented, signin.ErrSignInOff)
	case errors.Is(err, signin.ErrUnknownProvider):
		return nil, connect.NewError(connect.CodeInvalidArgument, signin.ErrUnknownProvider)
	case errors.Is(err, accounts.ErrNoAccount):
		return nil, connect.NewError(connect.CodeUnauthenticated, accounts.ErrNoAccount)
	case err != nil:
		return nil, fmt.Errorf("failed to start the sign-in: %w", err)
	}

	res := connect.NewResponse(&authv1.StartSignInResponse{AuthorizationUrl: out.AuthorizationURL})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", out.SetCookie)
	return res, nil
}

// intentOf reads only a link as a link: unset, as from every client before intents, signs in.
func intentOf(intent authv1.SignInIntent) accounts.Intent {
	if intent == authv1.SignInIntent_SIGN_IN_INTENT_LINK {
		return accounts.IntentLink
	}
	return accounts.IntentSignIn
}
