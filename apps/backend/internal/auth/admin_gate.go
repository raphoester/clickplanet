package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission/rpc_operators"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission/usecases/admit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission/usecases/admit_usecase/log_admit"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/guard/log_calls"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
)

type AdminConfig struct {
	Origin string

	RedirectURL string
	Google      signin.Client
	SessionTTL  time.Duration
}

func (c AdminConfig) withDefaults() AdminConfig {
	if c.SessionTTL <= 0 {
		c.SessionTTL = adminsignin.DefaultTTL
	}
	return c
}

// No origin is no admin page: no gate, and no admin sign-in.
func (c AdminConfig) Validate() error {
	if c.Origin == "" {
		return nil
	}

	var errs []error
	parsed, err := url.Parse(c.Origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.String() != parsed.Scheme+"://"+parsed.Host {
		errs = append(errs, fmt.Errorf("auth.admin.origin %q is not an origin: want scheme://host[:port], with no path and not \"*\"", c.Origin))
	}
	redirect, err := url.Parse(c.RedirectURL)
	if err != nil || !redirect.IsAbs() || redirect.RawQuery != "" {
		errs = append(errs, fmt.Errorf("auth.admin.redirectUrl %q is not an absolute URL without a query", c.RedirectURL))
	}
	if !c.Google.Configured() || c.Google.ClientSecret == "" {
		errs = append(errs, errors.New("auth.admin.google needs a clientId and a clientSecret: only Google opens an admin session"))
	}
	if c.SessionTTL < 0 {
		errs = append(errs, fmt.Errorf("auth.admin.sessionTTL %s is negative", c.SessionTTL))
	}

	return errors.Join(errs...)
}

var signInProcedures = []string{
	authv1connect.AdminServiceStartSignInProcedure,
	authv1connect.AdminServiceCompleteSignInProcedure,
	authv1connect.AdminServiceSignOutProcedure,
}

// Beside its sign-in, what only the admin page's session calls: an app cannot let another app in, and WhoAmI names
// the cookie's operator.
var sessionOnlyProcedures = []string{
	authv1connect.AdminServiceWhoAmIProcedure,
	authv1connect.AdminServiceDescribeConnectionProcedure,
	authv1connect.AdminServiceApproveConnectionProcedure,
}

// No origin is no admin page, and so no gate: an admin listener then refuses the boot.
func NewAdminGate(config AdminConfig, internalBindAddress string, logger *slog.Logger) (cpbootstrap.Gate, error) {
	if config.Origin == "" {
		return cpbootstrap.Gate{}, nil
	}
	if internalBindAddress == "" {
		return cpbootstrap.Gate{}, errors.New("the admin gate asks auth who operates over httpServer.internalBindAddress, which is empty")
	}

	auth := authv1connect.NewInternalServiceClient(&http.Client{Transport: http.DefaultTransport}, "http://"+internalBindAddress)
	admit := log_admit.New(
		admit_usecase.New(
			admission.NewPage(config.Origin, signInProcedures...).WithSessionOnly(sessionOnlyProcedures...),
			rpc_operators.New(auth),
		),
		logger,
	)

	return cpbootstrap.Gate{
		Middlewares:  []func(http.Handler) http.Handler{cphttpserver.NewCorsMiddleware(config.Origin)},
		Interceptors: []connect.Interceptor{guard.New(admit), log_calls.New(logger)},
	}, nil
}
