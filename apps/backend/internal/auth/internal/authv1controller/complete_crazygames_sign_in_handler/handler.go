package complete_crazygames_sign_in_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_crazygames_sign_in_usecase"
)

var ErrRefused = errors.New("CrazyGames did not sign this token")

var outcomes = map[accounts.Outcome]authv1.SignInOutcome{
	accounts.SignedIn: authv1.SignInOutcome_SIGN_IN_OUTCOME_SIGNED_IN,
	accounts.Linked:   authv1.SignInOutcome_SIGN_IN_OUTCOME_LINKED,
	accounts.Created:  authv1.SignInOutcome_SIGN_IN_OUTCOME_CREATED,
	accounts.Joined:   authv1.SignInOutcome_SIGN_IN_OUTCOME_SIGNED_IN,
}

type UseCase interface {
	Execute(ctx context.Context, in complete_crazygames_sign_in_usecase.In) (*complete_crazygames_sign_in_usecase.Out, error)
}

func New(useCase UseCase, logger *slog.Logger) CompleteCrazyGamesSignInHandler {
	return CompleteCrazyGamesSignInHandler{useCase: useCase, logger: logger}
}

type CompleteCrazyGamesSignInHandler struct {
	useCase UseCase
	logger  *slog.Logger
}

func (h CompleteCrazyGamesSignInHandler) CompleteCrazyGamesSignIn(
	ctx context.Context,
	req *connect.Request[authv1.CompleteCrazyGamesSignInRequest],
) (*connect.Response[authv1.CompleteCrazyGamesSignInResponse], error) {
	out, err := h.useCase.Execute(ctx, complete_crazygames_sign_in_usecase.In{
		Token:        req.Msg.GetToken(),
		CookieHeader: req.Header().Get("Cookie"),
		Landing:      arrivals.Landing{Tag: req.Msg.GetLanding().GetTag(), Referrer: req.Msg.GetLanding().GetReferrer()},
	})
	switch {
	case errors.Is(err, signin.ErrSignInOff), errors.Is(err, signin.ErrUnknownProvider):
		return nil, connect.NewError(connect.CodeUnimplemented, signin.ErrSignInOff)
	case errors.Is(err, signin.ErrProviderRefused):
		h.logger.Info("refused a CrazyGames sign-in", slog.String("procedure", req.Spec().Procedure), slog.Any("error", err))
		return nil, connect.NewError(connect.CodePermissionDenied, ErrRefused)
	case err != nil:
		return nil, fmt.Errorf("failed to complete the CrazyGames sign-in: %w", err)
	}

	res := connect.NewResponse(&authv1.CompleteCrazyGamesSignInResponse{AccountId: out.Account().String(), Outcome: outcomes[out.Outcome()]})
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Add("Set-Cookie", out.SetCookie())
	return res, nil
}
