// Package complete_sign_in_handler serves auth.v1.AuthService/CompleteSignIn.
package complete_sign_in_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_sign_in_usecase"
)

var (
	// ErrStartAgain is the whole of what a browser whose sign-in does not match is told.
	ErrStartAgain = errors.New("this sign-in cannot be completed: start again")
	// ErrRefused is the whole of what a browser the provider refused is told; the reason is logged.
	ErrRefused = errors.New("the provider did not sign you in")
)

var outcomes = map[accounts.Outcome]authv1.SignInOutcome{
	accounts.SignedIn: authv1.SignInOutcome_SIGN_IN_OUTCOME_SIGNED_IN,
	accounts.Linked:   authv1.SignInOutcome_SIGN_IN_OUTCOME_LINKED,
	accounts.Created:  authv1.SignInOutcome_SIGN_IN_OUTCOME_CREATED,
}

type UseCase interface {
	Execute(ctx context.Context, in complete_sign_in_usecase.In) (*complete_sign_in_usecase.Out, error)
}

func New(useCase UseCase, logger *slog.Logger) CompleteSignInHandler {
	return CompleteSignInHandler{useCase: useCase, logger: logger}
}

type CompleteSignInHandler struct {
	useCase UseCase
	logger  *slog.Logger
}

// CompleteSignIn clears the flow cookie on every answer it maps: a code is good once, so a flow is too.
func (h CompleteSignInHandler) CompleteSignIn(
	ctx context.Context,
	req *connect.Request[authv1.CompleteSignInRequest],
) (*connect.Response[authv1.CompleteSignInResponse], error) {
	out, err := h.useCase.Execute(ctx, complete_sign_in_usecase.In{
		Code:         req.Msg.GetCode(),
		State:        req.Msg.GetState(),
		CookieHeader: req.Header().Get("Cookie"),
	})
	switch {
	case errors.Is(err, signin.ErrSignInOff):
		return nil, connect.NewError(connect.CodeUnimplemented, signin.ErrSignInOff)
	case errors.Is(err, signin.ErrFlowInvalid):
		h.logRefusal(req, err)
		return nil, clearingFlow(connect.NewError(connect.CodeFailedPrecondition, ErrStartAgain))
	case errors.Is(err, signin.ErrProviderRefused):
		h.logRefusal(req, err)
		return nil, clearingFlow(connect.NewError(connect.CodePermissionDenied, ErrRefused))
	case err != nil:
		return nil, fmt.Errorf("failed to complete the sign-in: %w", err)
	}

	res := connect.NewResponse(&authv1.CompleteSignInResponse{AccountId: out.Account.String(), Outcome: outcomes[out.Outcome]})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", out.SetCookie)
	res.Header().Add("Set-Cookie", signin.ClearFlowCookie())
	return res, nil
}

// logRefusal is Info, not the error net's Error: a stale tab or a cancelled consent is the common case.
func (h CompleteSignInHandler) logRefusal(req *connect.Request[authv1.CompleteSignInRequest], err error) {
	h.logger.Info("refused a sign-in", slog.String("procedure", req.Spec().Procedure), slog.Any("error", err))
}

func clearingFlow(err *connect.Error) *connect.Error {
	err.Meta().Add("Set-Cookie", signin.ClearFlowCookie())
	return err
}
