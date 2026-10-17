package admin_complete_sign_in_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin/usecases/complete_admin_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

var (
	ErrStartAgain = errors.New("this sign-in cannot be completed: start again")
	ErrRefused    = errors.New("this Google account cannot sign in to the admin page")
)

type UseCase interface {
	Execute(ctx context.Context, in complete_admin_sign_in_usecase.In) (*complete_admin_sign_in_usecase.Out, error)
}

func New(useCase UseCase, logger *slog.Logger) AdminCompleteSignInHandler {
	return AdminCompleteSignInHandler{useCase: useCase, logger: logger}
}

type AdminCompleteSignInHandler struct {
	useCase UseCase
	logger  *slog.Logger
}

func (h AdminCompleteSignInHandler) CompleteSignIn(
	ctx context.Context,
	req *connect.Request[authv1.AdminCompleteSignInRequest],
) (*connect.Response[authv1.AdminCompleteSignInResponse], error) {
	out, err := h.useCase.Execute(ctx, complete_admin_sign_in_usecase.In{
		Code:         req.Msg.GetCode(),
		State:        req.Msg.GetState(),
		CookieHeader: req.Header().Get("Cookie"),
	})
	switch {
	case errors.Is(err, adminsignin.ErrSignInOff):
		return nil, connect.NewError(connect.CodeUnimplemented, adminsignin.ErrSignInOff)
	case errors.Is(err, signin.ErrFlowInvalid):
		h.logRefusal(err)
		return nil, refusal(connect.CodeFailedPrecondition, ErrStartAgain)
	case errors.Is(err, signin.ErrProviderRefused), errors.Is(err, adminsignin.ErrNotAnOperator):
		h.logRefusal(err)
		return nil, refusal(connect.CodePermissionDenied, ErrRefused)
	case err != nil:
		return nil, fmt.Errorf("failed to complete the admin sign-in: %w", err)
	}

	res := connect.NewResponse(&authv1.AdminCompleteSignInResponse{AccountId: out.Account.String()})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", out.SetCookie)
	res.Header().Add("Set-Cookie", adminsignin.ExpiredFlowCookie())
	return res, nil
}

func (h AdminCompleteSignInHandler) logRefusal(err error) {
	h.logger.Warn("refused an admin sign-in", slog.Any("error", err))
}

func refusal(code connect.Code, reason error) *connect.Error {
	refused := connect.NewError(code, reason)
	refused.Meta().Add("Set-Cookie", adminsignin.ExpiredFlowCookie())
	return refused
}
