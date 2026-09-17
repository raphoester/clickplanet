// Package cpbootstrap runs a set of bounded contexts as one process.
//
// A module is handed a set of registrars and nothing else: it mounts its RPC
// routes, registers whatever has to keep running, and says what has to be shut
// down. It never sees the router, the server, the signal handler or another
// module's dependencies, so what two contexts share is exactly what the
// composition root chose to hand both of them — see cmd/api.
//
// This is a modular monolith and is meant to stay one: there is one binary, one
// port and one process, and a module is a boundary inside it rather than a
// thing that could be deployed on its own.
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

// Module is one bounded context and the sequence that builds it.
type Module struct {
	Name string

	// Off is skipped: its routes are never mounted, so they 404 rather than existing and refusing.
	Enabled bool

	// DiSequence builds the context. Its ctx is the startup one and is
	// cancelled once every module is built, so nothing may capture it — what
	// outlives startup is registered on Props.Runners, which is handed the
	// process-lifetime context instead.
	DiSequence func(ctx context.Context, props Props) error
}

// Props is everything a module may reach outside itself.
type Props struct {
	Logger  *slog.Logger
	Metrics prometheus.Registerer

	// The transport every module answers over, and the only config a module
	// reads that is not its own.
	Server ServerConfig

	RPC     RPCRegistrar
	Runners RunnerRegistrar
	Closers CloserRegistrar

	// AdminRPC mounts on the loopback admin listener, which has no authentication and no CORS.
	AdminRPC RPCRegistrar

	// InternalRPC mounts what other modules call, on the loopback internal listener; Internal reaches it.
	InternalRPC RPCRegistrar
	Internal    InternalDialer

	// Events is what a module publishes for others to hear, and subscribes to with Subscribe.
	Events EventBus
}

// InternalDialer is how a module calls another: over the internal listener, never in its own stack trace.
type InternalDialer interface {
	Dial() (connect.HTTPClient, string, error)
}

// RPCRegistrar mounts a Connect service.
//
// A module hands over what builds the handler rather than the handler itself,
// because a Connect interceptor is baked in at construction: there is no way to
// wrap one afterwards, and an HTTP middleware is too late — by then the error is
// already a serialized response body. Building here is therefore the only way
// the server can guarantee something around every procedure in the process.
//
// What it guarantees is the error net: no handler's raw error reaches the wire,
// whether or not the module that wrote it remembered to ask.
type RPCRegistrar interface {
	Mount(build ServiceBuilder, interceptors ...connect.Interceptor) error
}

// ServiceBuilder is a generated New<Service>Handler with its service bound. A
// module writes the closure, because the generated constructor takes the
// service interface while the module holds the concrete type — which is a
// conversion no type parameter can infer:
//
//	return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
//		return planetv1connect.NewClickServiceHandler(service, options...)
//	}, interceptors...)
type ServiceBuilder func(options ...connect.HandlerOption) (string, http.Handler)

// RunnerRegistrar takes a goroutine that runs until its context is cancelled.
type RunnerRegistrar interface {
	Add(runner Runner)
}

// Runner is a loop that lives as long as the process, and names itself.
type Runner interface {
	Name() string
	Run(ctx context.Context)
}

// CloserRegistrar takes a cleanup, run in reverse registration order before the
// runners are waited on.
type CloserRegistrar interface {
	Add(name string, close func() error)
}

// ServerConfig is the transport, which the modules speak over but none of them owns.
type ServerConfig struct {
	BindAddress string

	// Must stay well under the proxy's idle cut: Cloudflare answers 524 at ~125s.
	StreamHeartbeat time.Duration

	// Empty serves no admin listener; anything but a loopback address refuses the boot.
	AdminBindAddress string

	// Empty serves no internal listener, and a module that dials it refuses the boot; anything but loopback refuses it too.
	InternalBindAddress string

	// The frontend's origin, exactly: scheme, host and port. The only origin a
	// browser may call from, with credentials. Empty or "*" refuses the boot.
	AllowedOrigin string
}

// Validate refuses the address that has no usable zero value: empty listens on port 80.
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

// validateOrigin refuses what a browser would refuse later, one request at a
// time: a credentialed answer must name one origin, and an origin has no path.
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

	// How long the whole DI sequence may take before the boot is abandoned.
	StartupTimeout time.Duration

	// How long the HTTP server is given to drain in-flight requests.
	ShutdownTimeout time.Duration

	Logger *slog.Logger

	// Built in the order they are given, and that order is the only coupling
	// between them: no module reads what another one left behind, because
	// there is nowhere to leave it.
	Modules []Module
}

const (
	defaultStartupTimeout  = 5 * time.Second
	defaultShutdownTimeout = 5 * time.Second
)

// Run builds every module, then serves until the process is signalled.
func Run(ctx context.Context, options Options) error {
	options = options.withDefaults()

	if !slices.ContainsFunc(options.Modules, func(m Module) bool { return m.Enabled }) {
		return errors.New("the process serves nothing: every module it was given is off")
	}

	metrics := cpprom.NewRegistry()
	errorNet := cpconnect.NewErrorInterceptor(options.Logger, nil)

	// Cancelled when shutdown starts, and it ends every open stream.
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
	// Before any runner starts: a subscriber registered later could miss what was published at boot.
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

// buildModules runs every module's DI sequence under one startup deadline.
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
