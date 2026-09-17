// Package create_session_handler serves auth.v1.AuthService/CreateSession.
package create_session_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/create_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// ErrRefused is the whole of what a refused caller is told; the reason is logged.
var ErrRefused = errors.New("could not start a session")

type UseCase interface {
	Execute(ctx context.Context, in create_session_usecase.In) (*create_session_usecase.Out, error)
}

func New(useCase UseCase, logger *slog.Logger) CreateSessionHandler {
	return CreateSessionHandler{useCase: useCase, logger: logger}
}

type CreateSessionHandler struct {
	useCase UseCase
	logger  *slog.Logger
}

// CreateSession answers no-store: a token and its cookie belong to one caller at one address.
func (h CreateSessionHandler) CreateSession(
	ctx context.Context,
	req *connect.Request[authv1.CreateSessionRequest],
) (*connect.Response[authv1.CreateSessionResponse], error) {
	out, err := h.useCase.Execute(ctx, create_session_usecase.In{
		AttestationToken: req.Msg.GetAttestationToken(),
		IP:               cpctx.GetSourceIP(ctx),
		CookieHeader:     req.Header().Get("Cookie"),
	})
	if errors.Is(err, attestation.ErrAttestationFailed) {
		// Info, not the error net's Error: a refusal is the check doing its job.
		h.logger.Info("refused a session", slog.String("procedure", req.Spec().Procedure), slog.Any("error", err))
		return nil, connect.NewError(connect.CodePermissionDenied, ErrRefused)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create a session: %w", err)
	}

	res := connect.NewResponse(&authv1.CreateSessionResponse{
		Token:           out.Token.Value,
		ExpiresAtUnixMs: out.Token.ExpiresAt.UnixMilli(),
	})
	res.Header().Set("Cache-Control", "no-store")
	if out.SetCookie != "" {
		res.Header().Add("Set-Cookie", out.SetCookie)
	}

	return res, nil
}
