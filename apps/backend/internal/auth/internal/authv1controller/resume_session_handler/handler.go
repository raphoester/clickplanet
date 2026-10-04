package resume_session_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resume_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var ErrRefused = errors.New("could not resume a session")

type UseCase interface {
	Execute(ctx context.Context, in resume_session_usecase.In) (*resume_session_usecase.Out, error)
}

func New(useCase UseCase) ResumeSessionHandler {
	return ResumeSessionHandler{useCase: useCase}
}

type ResumeSessionHandler struct {
	useCase UseCase
}

func (h ResumeSessionHandler) ResumeSession(
	ctx context.Context,
	req *connect.Request[authv1.ResumeSessionRequest],
) (*connect.Response[authv1.ResumeSessionResponse], error) {
	out, err := h.useCase.Execute(ctx, resume_session_usecase.In{
		IP:           cpctx.GetSourceIP(ctx),
		CookieHeader: req.Header().Get("Cookie"),
	})
	if errors.Is(err, resume_session_usecase.ErrNoAddress) {
		return nil, connect.NewError(connect.CodePermissionDenied, ErrRefused)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to resume a session: %w", err)
	}

	answer := &authv1.ResumeSessionResponse{}
	if out.Token != nil {
		answer.Token = out.Token.Value
		answer.ExpiresAtUnixMs = out.Token.ExpiresAt.UnixMilli()
	}
	res := connect.NewResponse(answer)
	res.Header().Set("Cache-Control", "no-store")
	if out.SetCookie != "" {
		res.Header().Add("Set-Cookie", out.SetCookie)
	}
	return res, nil
}
