package authv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_email_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/create_session_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/delete_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_sign_in_options_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_verifying_key_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_everywhere_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_email_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_sign_in_handler"
)

type AuthService struct {
	create_session_handler.CreateSessionHandler
	get_me_handler.GetMeHandler
	get_sign_in_options_handler.GetSignInOptionsHandler
	start_sign_in_handler.StartSignInHandler
	complete_sign_in_handler.CompleteSignInHandler
	start_email_sign_in_handler.StartEmailSignInHandler
	complete_email_sign_in_handler.CompleteEmailSignInHandler
	sign_out_handler.SignOutHandler
	sign_out_everywhere_handler.SignOutEverywhereHandler
	delete_account_handler.DeleteAccountHandler
}

var _ authv1connect.AuthServiceHandler = AuthService{}

type InternalService struct {
	get_verifying_key_handler.GetVerifyingKeyHandler
	get_account_handler.GetAccountHandler
}

var _ authv1connect.InternalServiceHandler = InternalService{}
