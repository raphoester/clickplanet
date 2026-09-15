package sessionv1controller

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	sessionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain/session_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type SessionService struct {
	service session_service.IService

	// Only a refusal is logged here, and only because it is not an error — see
	// toConnect. Everything else is the error net's.
	logger *slog.Logger
}

var _ sessionv1connect.SessionServiceHandler = (*SessionService)(nil)

func NewSessionService(service session_service.IService, logger *slog.Logger) *SessionService {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return &SessionService{service: service, logger: logger}
}

func (s *SessionService) CreateSession(
	ctx context.Context,
	req *connect.Request[sessionv1.CreateSessionRequest],
) (*connect.Response[sessionv1.CreateSessionResponse], error) {
	token, err := s.service.Create(ctx, req.Msg.GetAttestationToken(), cpctx.GetSourceIP(ctx))
	if err != nil {
		return nil, toConnect(s.logger, req.Spec().Procedure, err)
	}

	res := connect.NewResponse(&sessionv1.CreateSessionResponse{
		Token:           token.Value,
		ExpiresAtUnixMs: token.ExpiresAt.UnixMilli(),
	})

	// A minted token belongs to one caller and one address. Nothing in front of
	// this may hold onto it, whatever it does with the other routes.
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
