package cpbootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpprom"
)

type Module struct {
	Name string

	Enabled bool

	DiSequence func(ctx context.Context, props Props) error
}

type Props struct {
	Logger  *slog.Logger
	Metrics prometheus.Registerer

	Server ServerConfig

	RPC     RPCRegistrar
	Runners RunnerRegistrar
	Closers CloserRegistrar

	AdminRPC RPCRegistrar

	InternalRPC RPCRegistrar
	Internal    InternalDialer

	Events EventBus
}

type InternalDialer interface {
	Dial() (connect.HTTPClient, string, error)
}

type RPCRegistrar interface {
	Mount(build ServiceBuilder, interceptors ...connect.Interceptor) error
}

type ServiceBuilder func(options ...connect.HandlerOption) (string, http.Handler)

type RunnerRegistrar interface {
	Add(runner Runner)
}

type Runner interface {
	Name() string
	Run(ctx context.Context)
}

type CloserRegistrar interface {
	Add(name string, close func() error)
}

type ServerConfig struct {
	BindAddress string

	StreamHeartbeat time.Duration

	AdminBindAddress string

	InternalBindAddress string

	AllowedOrigin string
}

func (c ServerConfig) Validate() error {
	if c.BindAddress == "" {
		return errors.New("httpServer.bindAddress is empty")
	}

	if err := validateOrigin(c.AllowedOrigin); err != nil {
		return err
	}

	if c.AdminBindAddress != "" && !isLoopback(c.AdminBindAddress) {
		return fmt.Errorf(
			"httpServer.adminBindAddress %q is not a loopback host:port: the admin services have no authentication",
			c.AdminBindAddress,
		)
	}

	if c.InternalBindAddress != "" && !isLoopback(c.InternalBindAddress) {
		return fmt.Errorf(
			"httpServer.internalBindAddress %q is not a loopback host:port: its callers are trusted",
			c.InternalBindAddress,
		)
	}

	return nil
}

func validateOrigin(origin string) error {
	if origin == "" {
		return errors.New("httpServer.allowedOrigin is empty: set it to the frontend's origin")
	}

	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.String() != parsed.Scheme+"://"+parsed.Host {
		return fmt.Errorf(
			"httpServer.allowedOrigin %q is not an origin: want scheme://host[:port], with no path and not \"*\"",
			origin,
		)
	}

	return nil
}

func isLoopback(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type Options struct {
	Server ServerConfig

	StartupTimeout time.Duration

	ShutdownTimeout time.Duration

	Logger *slog.Logger

	Modules []Module
}

const (
	defaultStartupTimeout  = 5 * time.Second
	defaultShutdownTimeout = 5 * time.Second
)

func Run(ctx context.Context, options Options) error {
	options = options.withDefaults()

	if !slices.ContainsFunc(options.Modules, func(m Module) bool { return m.Enabled }) {
		return errors.New("the process serves nothing: every module it was given is off")
	}

	metrics := cpprom.NewRegistry()
	errorNet := cpconnect.NewErrorInterceptor(options.Logger, nil)

	draining, drain := context.WithCancel(context.Background())
	defer drain()

	drainNet := newDrainInterceptor(draining)
	routes := newRPCRoutes(errorNet, drainNet)
	adminRoutes := newRPCRoutes(errorNet, drainNet)
	internalRoutes := newRPCRoutes(errorNet, drainNet)
	runners := newRunnerRegistry()
	closers := newCloserRegistry()
	events := newEventBus(metrics)

	registrars := registrars{routes: routes, admin: adminRoutes, internal: internalRoutes, events: events}
	if err := buildModules(ctx, options, metrics, registrars, runners, closers); err != nil {
		return err
	}
	// Before any runner starts: a subscriber registered later could miss boot events.
	events.seal()

	router := http.NewServeMux()
	routes.mountOn(router, cphttpserver.MiddlewareStack(
		cphttpserver.NewLoggingMiddleware(options.Logger),
		cphttpserver.IPReaderMiddleware,
		cphttpserver.NewCorsMiddleware(options.Server.AllowedOrigin),
	))
	mountMetrics(router, metrics, options.Logger)

	loopbacks, err := listenLoopbacks(options, adminRoutes, internalRoutes)
	if err != nil {
		return err
	}

	return serve(ctx, options, router, loopbacks, drain, runners, closers)
}

func listenLoopbacks(options Options, adminRoutes, internalRoutes *rpcRoutes) ([]*loopbackServer, error) {
	admin, err := listenLoopback(options, "admin", "adminBindAddress", options.Server.AdminBindAddress, adminRoutes)
	if err != nil {
		return nil, err
	}

	internal, err := listenLoopback(options, "internal", "internalBindAddress", options.Server.InternalBindAddress, internalRoutes)
	if err != nil {
		if admin != nil {
			_ = admin.listener.Close()
		}
		return nil, err
	}

	var loopbacks []*loopbackServer
	for _, loopback := range []*loopbackServer{admin, internal} {
		if loopback != nil {
			loopbacks = append(loopbacks, loopback)
		}
	}

	return loopbacks, nil
}

type registrars struct {
	routes   *rpcRoutes
	admin    *rpcRoutes
	internal *rpcRoutes
	events   *eventBus
}

func buildModules(
	ctx context.Context,
	options Options,
	metrics *prometheus.Registry,
	registrars registrars,
	runners *runnerRegistry,
	closers *closerRegistry,
) error {
	ctx, cancel := context.WithTimeout(ctx, options.StartupTimeout)
	defer cancel()

	for _, module := range options.Modules {
		if !module.Enabled {
			options.Logger.Info("module disabled", slog.String("module", module.Name))
			continue
		}

		before := runners.count()

		err := module.DiSequence(ctx, Props{
			Logger:      options.Logger,
			Metrics:     metrics,
			Server:      options.Server,
			RPC:         registrars.routes.forModule(module.Name),
			AdminRPC:    registrars.admin.forModule(module.Name),
			InternalRPC: registrars.internal.forModule(module.Name),
			Internal:    internalDialer{address: options.Server.InternalBindAddress},
			Events:      registrars.events,
			Runners:     runners,
			Closers:     closers,
		})
		if err != nil {
			return fmt.Errorf("failed to build the %s module: %w", module.Name, err)
		}

		options.Logger.Debug("module built",
			slog.String("module", module.Name),
			slog.Int("runners", runners.count()-before),
		)
	}

	return nil
}

func (o Options) withDefaults() Options {
	if o.StartupTimeout <= 0 {
		o.StartupTimeout = defaultStartupTimeout
	}
	if o.ShutdownTimeout <= 0 {
		o.ShutdownTimeout = defaultShutdownTimeout
	}
	return o
}
