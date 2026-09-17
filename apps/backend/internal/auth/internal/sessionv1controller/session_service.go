// Package sessionv1controller serves session.v1.SessionService, which mints a click token with no account.
//
// Deprecated: auth.v1.AuthService/CreateSession replaces it. It goes once no client calls it.
package sessionv1controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	sessionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/create_anonymous_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// ErrRefused is the whole of what a refused caller is told; the reason is logged.
var ErrRefused = errors.New("could not start a session")

type UseCase interface {
	Execute(ctx context.Context, in create_anonymous_session_usecase.In) (*cpsession.Token, error)
}

type SessionService struct {
	useCase UseCase
	logger  *slog.Logger
}

var _ sessionv1connect.SessionServiceHandler = SessionService{}

func NewSessionService(useCase UseCase, logger *slog.Logger) SessionService {
	return SessionService{useCase: useCase, logger: logger}
}

func (s SessionService) CreateSession(
	ctx context.Context,
	req *connect.Request[sessionv1.CreateSessionRequest],
) (*connect.Response[sessionv1.CreateSessionResponse], error) {
	token, err := s.useCase.Execute(ctx, create_anonymous_session_usecase.In{
		AttestationToken: req.Msg.GetAttestationToken(),
		IP:               cpctx.GetSourceIP(ctx),
	})
	if errors.Is(err, attestation.ErrAttestationFailed) {
		s.logger.Info("refused a session", slog.String("procedure", req.Spec().Procedure), slog.Any("error", err))
		return nil, connect.NewError(connect.CodePermissionDenied, ErrRefused)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create a session: %w", err)
	}

	res := connect.NewResponse(&sessionv1.CreateSessionResponse{
		Token:           token.Value,
		ExpiresAtUnixMs: token.ExpiresAt.UnixMilli(),
	})
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
