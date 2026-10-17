package admingate

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/admission"
	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/admission/rpc_auth_operators"
	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/admission/usecases/admit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/admission/usecases/admit_usecase/log_admit"
	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/guard/log_calls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
)

type Config struct {
	Origin string
}

func (c Config) Validate() error {
	if c.Origin == "" {
		return nil
	}

	parsed, err := url.Parse(c.Origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.String() != parsed.Scheme+"://"+parsed.Host {
		return fmt.Errorf("adminGate.origin %q is not an origin: want scheme://host[:port], with no path and not \"*\"", c.Origin)
	}

	return nil
}

// No origin is no admin page, and so no gate: a gated admin listener then refuses the boot.
func New(config Config, internalBindAddress string, logger *slog.Logger) (cpbootstrap.Gate, error) {
	if config.Origin == "" {
		return cpbootstrap.Gate{}, nil
	}
	if internalBindAddress == "" {
		return cpbootstrap.Gate{}, errors.New("adminGate asks auth who operates over httpServer.internalBindAddress, which is empty")
	}

	auth := authv1connect.NewInternalServiceClient(&http.Client{Transport: http.DefaultTransport}, "http://"+internalBindAddress)
	admit := log_admit.New(
		admit_usecase.New(admission.NewPage(config.Origin), rpc_auth_operators.New(auth)),
		logger,
	)

	return cpbootstrap.Gate{
		Middlewares:  []func(http.Handler) http.Handler{cphttpserver.NewCorsMiddleware(config.Origin)},
		Interceptors: []connect.Interceptor{guard.New(admit), log_calls.New(logger)},
	}, nil
}
