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
	"net/http"
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
	Add(name string, run func(ctx context.Context))
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
}

// Validate refuses the address that has no usable zero value: empty listens on port 80.
func (c ServerConfig) Validate() error {
	if c.BindAddress == "" {
		return errors.New("httpServer.bindAddress is empty")
	}

	return nil
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
	routes := newRPCRoutes(cpconnect.NewErrorInterceptor(options.Logger, nil))
	runners := newRunnerRegistry()
	closers := newCloserRegistry()

	if err := buildModules(ctx, options, metrics, routes, runners, closers); err != nil {
		return err
	}

	router := http.NewServeMux()
	routes.mountOn(router, cphttpserver.MiddlewareStack(
		cphttpserver.NewLoggingMiddleware(options.Logger),
		cphttpserver.IPReaderMiddleware,
		cphttpserver.CorsMiddleware,
	))
	mountMetrics(router, metrics, options.Logger)

	return serve(ctx, options, router, runners, closers)
}

// buildModules runs every module's DI sequence under one startup deadline.
func buildModules(
	ctx context.Context,
	options Options,
	metrics *prometheus.Registry,
	routes *rpcRoutes,
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
			Logger:  options.Logger,
			Metrics: metrics,
			Server:  options.Server,
			RPC:     routes.forModule(module.Name),
			Runners: runners,
			Closers: closers,
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
