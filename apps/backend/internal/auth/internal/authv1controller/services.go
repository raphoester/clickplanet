package authv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/create_session_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
)

// AuthService is the handlers in a bag, for the generated handler.
type AuthService struct {
	create_session_handler.CreateSessionHandler
	get_me_handler.GetMeHandler
}

var _ authv1connect.AuthServiceHandler = AuthService{}
