package authv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/create_session_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/delete_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_sign_in_options_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_verifying_key_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_everywhere_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_sign_in_handler"
)

// AuthService is the handlers in a bag, for the generated handler.
type AuthService struct {
	create_session_handler.CreateSessionHandler
	get_me_handler.GetMeHandler
	get_sign_in_options_handler.GetSignInOptionsHandler
	start_sign_in_handler.StartSignInHandler
	complete_sign_in_handler.CompleteSignInHandler
	sign_out_handler.SignOutHandler
	sign_out_everywhere_handler.SignOutEverywhereHandler
	delete_account_handler.DeleteAccountHandler
}

var _ authv1connect.AuthServiceHandler = AuthService{}

// InternalService is what other modules ask, on the loopback listener only.
type InternalService struct {
	get_verifying_key_handler.GetVerifyingKeyHandler
}

var _ authv1connect.InternalServiceHandler = InternalService{}
