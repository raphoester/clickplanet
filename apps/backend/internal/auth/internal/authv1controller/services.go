package authv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/resolve_account_handler"
)

// AuthService is the public handlers in a bag, for the generated handler.
type AuthService struct {
	get_me_handler.GetMeHandler
}

var _ authv1connect.AuthServiceHandler = AuthService{}

// InternalService is what other modules call, on the internal listener only.
type InternalService struct {
	resolve_account_handler.ResolveAccountHandler
}

var _ authv1connect.InternalServiceHandler = InternalService{}
