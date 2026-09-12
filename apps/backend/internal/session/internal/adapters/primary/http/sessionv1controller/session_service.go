package sessionv1controller

import (
	"context"

	"connectrpc.com/connect"

	sessionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain/session_service"
)

type SessionService struct {
	service session_service.IService
}

var _ sessionv1connect.SessionServiceHandler = (*SessionService)(nil)

func NewSessionService(service session_service.IService) *SessionService {
	return &SessionService{service: service}
}

func (s *SessionService) CreateSession(
	ctx context.Context,
	req *connect.Request[sessionv1.CreateSessionRequest],
) (*connect.Response[sessionv1.CreateSessionResponse], error) {
	token, err := s.service.Create(ctx, req.Msg.GetAttestationToken(), ctxutil.GetSourceIP(ctx))
	if err != nil {
		return nil, err
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
