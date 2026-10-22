package authv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/admin_complete_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/admin_sign_out_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/admin_start_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_email_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/create_session_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/delete_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_accounts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_operator_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_sign_in_options_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_token_operator_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_verifying_key_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/resume_session_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_everywhere_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_email_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/who_am_i_handler"
)

type AuthService struct {
	create_session_handler.CreateSessionHandler
	resume_session_handler.ResumeSessionHandler
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
	get_accounts_handler.GetAccountsHandler
	get_operator_handler.GetOperatorHandler
	get_token_operator_handler.GetTokenOperatorHandler
}

var _ authv1connect.InternalServiceHandler = InternalService{}

type AdminService struct {
	admin_start_sign_in_handler.AdminStartSignInHandler
	admin_complete_sign_in_handler.AdminCompleteSignInHandler
	admin_sign_out_handler.AdminSignOutHandler
	who_am_i_handler.WhoAmIHandler
}

var _ authv1connect.AdminServiceHandler = AdminService{}
