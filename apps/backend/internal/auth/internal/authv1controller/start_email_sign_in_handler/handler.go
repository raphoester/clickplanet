package start_email_sign_in_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/authprovider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_email_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var ErrRefused = errors.New("could not send a code")

type UseCase interface {
	Execute(ctx context.Context, in start_email_sign_in_usecase.In) (*start_email_sign_in_usecase.Out, error)
}

func New(useCase UseCase, logger *slog.Logger) StartEmailSignInHandler {
	return StartEmailSignInHandler{useCase: useCase, logger: logger}
}

type StartEmailSignInHandler struct {
	useCase UseCase
	logger  *slog.Logger
}

func (h StartEmailSignInHandler) StartEmailSignIn(
	ctx context.Context,
	req *connect.Request[authv1.StartEmailSignInRequest],
) (*connect.Response[authv1.StartEmailSignInResponse], error) {
	out, err := h.useCase.Execute(ctx, start_email_sign_in_usecase.In{
		Address:          req.Msg.GetEmail(),
		Intent:           authprovider.IntentOf(req.Msg.GetIntent()),
		AttestationToken: req.Msg.GetAttestationToken(),
		IP:               cpctx.GetSourceIP(ctx),
		CookieHeader:     req.Header().Get("Cookie"),
	})
	switch {
	case errors.Is(err, signin.ErrSignInOff):
		return nil, connect.NewError(connect.CodeUnimplemented, signin.ErrSignInOff)
	case errors.Is(err, signin.ErrAddressInvalid):
		return nil, emailRefusal(signin.ErrAddressInvalid, authv1.EmailRefusalReason_EMAIL_REFUSAL_REASON_INVALID)
	case errors.Is(err, signin.ErrAddressDisposable):
		return nil, emailRefusal(signin.ErrAddressDisposable, authv1.EmailRefusalReason_EMAIL_REFUSAL_REASON_DISPOSABLE)
	case errors.Is(err, attestation.ErrAttestationFailed):
		h.logger.Info("refused an email sign-in", slog.String("procedure", req.Spec().Procedure), slog.Any("error", err))
		return nil, connect.NewError(connect.CodePermissionDenied, ErrRefused)
	case errors.Is(err, accounts.ErrNoAccount):
		return nil, connect.NewError(connect.CodeUnauthenticated, accounts.ErrNoAccount)
	case errors.Is(err, signin.ErrTooManyCodes):
		return nil, connect.NewError(connect.CodeResourceExhausted, signin.ErrTooManyCodes)
	case err != nil:
		return nil, fmt.Errorf("failed to start the email sign-in: %w", err)
	}

	res := connect.NewResponse(&authv1.StartEmailSignInResponse{})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", out.SetCookie)
	return res, nil
}

func emailRefusal(sentinel error, reason authv1.EmailRefusalReason) *connect.Error {
	refused := connect.NewError(connect.CodeInvalidArgument, sentinel)
	if detail, err := connect.NewErrorDetail(&authv1.EmailRefusal{Reason: reason}); err == nil {
		refused.AddDetail(detail)
	}
	return refused
}
