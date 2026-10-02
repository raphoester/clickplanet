package complete_email_sign_in_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_email_sign_in_usecase"
)

var ErrStartAgain = errors.New("this sign-in cannot be completed: ask for a new code")

var outcomes = map[accounts.Outcome]authv1.SignInOutcome{
	accounts.SignedIn: authv1.SignInOutcome_SIGN_IN_OUTCOME_SIGNED_IN,
	accounts.Linked:   authv1.SignInOutcome_SIGN_IN_OUTCOME_LINKED,
	accounts.Created:  authv1.SignInOutcome_SIGN_IN_OUTCOME_CREATED,
	accounts.Joined:   authv1.SignInOutcome_SIGN_IN_OUTCOME_SIGNED_IN,
}

type UseCase interface {
	Execute(ctx context.Context, in complete_email_sign_in_usecase.In) (*complete_email_sign_in_usecase.Out, error)
}

func New(useCase UseCase, logger *slog.Logger) CompleteEmailSignInHandler {
	return CompleteEmailSignInHandler{useCase: useCase, logger: logger}
}

type CompleteEmailSignInHandler struct {
	useCase UseCase
	logger  *slog.Logger
}

func (h CompleteEmailSignInHandler) CompleteEmailSignIn(
	ctx context.Context,
	req *connect.Request[authv1.CompleteEmailSignInRequest],
) (*connect.Response[authv1.CompleteEmailSignInResponse], error) {
	out, err := h.useCase.Execute(ctx, complete_email_sign_in_usecase.In{
		Code:         req.Msg.GetCode(),
		CookieHeader: req.Header().Get("Cookie"),
	})
	switch {
	case errors.Is(err, signin.ErrSignInOff):
		return nil, connect.NewError(connect.CodeUnimplemented, signin.ErrSignInOff)
	case errors.Is(err, signin.ErrWrongCode):
		return nil, connect.NewError(connect.CodeInvalidArgument, signin.ErrWrongCode)
	case errors.Is(err, signin.ErrFlowInvalid):
		h.logRefusal(req, err)
		return nil, refusal(connect.CodeFailedPrecondition, ErrStartAgain)
	case errors.Is(err, accounts.ErrIdentityLinkedElsewhere):
		h.logRefusal(req, err)
		return nil, linkRefusal(accounts.ErrIdentityLinkedElsewhere, authv1.LinkRefusalReason_LINK_REFUSAL_REASON_IDENTITY_LINKED_ELSEWHERE)
	case errors.Is(err, accounts.ErrProviderAlreadyLinked):
		h.logRefusal(req, err)
		return nil, linkRefusal(accounts.ErrProviderAlreadyLinked, authv1.LinkRefusalReason_LINK_REFUSAL_REASON_PROVIDER_ALREADY_LINKED)
	case err != nil:
		return nil, fmt.Errorf("failed to complete the email sign-in: %w", err)
	}

	res := connect.NewResponse(&authv1.CompleteEmailSignInResponse{AccountId: out.Account.String(), Outcome: outcomes[out.Outcome]})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", out.SetCookie)
	res.Header().Add("Set-Cookie", signin.ExpiredChallengeCookie())
	return res, nil
}

func (h CompleteEmailSignInHandler) logRefusal(req *connect.Request[authv1.CompleteEmailSignInRequest], err error) {
	h.logger.Info("refused an email sign-in", slog.String("procedure", req.Spec().Procedure), slog.Any("error", err))
}

func linkRefusal(sentinel error, reason authv1.LinkRefusalReason) *connect.Error {
	refused := refusal(connect.CodeAlreadyExists, sentinel)
	if detail, err := connect.NewErrorDetail(&authv1.LinkRefusal{Reason: reason}); err == nil {
		refused.AddDetail(detail)
	}
	return refused
}

func refusal(code connect.Code, reason error) *connect.Error {
	refused := connect.NewError(code, reason)
	refused.Meta().Add("Set-Cookie", signin.ExpiredChallengeCookie())
	return refused
}
