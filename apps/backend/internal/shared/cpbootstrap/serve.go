package cpbootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/metrics/v1/metricsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpprom"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func mountMetrics(router *http.ServeMux, metrics *prometheus.Registry, logger *slog.Logger) {
	metricsRouter := http.NewServeMux()
	metricsRouter.HandleFunc("GET /", cpprom.HandlerForRegistry(metrics).ServeHTTP)

	router.Handle("/metrics", cphttpserver.MiddlewareStack(
		cphttpserver.NewLoggingMiddleware(logger),
	)(metricsRouter))
}

// The admin page's read of /metrics, on the admin routes: /metrics itself is on no listener Caddy reaches.
func mountMetricsAdmin(routes *rpcRoutes, metrics *prometheus.Registry) error {
	service := cpprom.NewAdminService(metrics, time.Now(), cptime.SystemClock{})
	return routes.forModule("metrics").Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return metricsv1connect.NewAdminServiceHandler(service, options...)
	})
}

type sideServer struct {
	name     string
	server   *http.Server
	listener net.Listener
}

func listenLoopback(options Options, listen listenFunc, name, key, address string, routes *rpcRoutes) (*sideServer, error) {
	if address == "" {
		if len(routes.paths) > 0 {
			options.Logger.Info(name+" listener off, its services not served", slog.Int("services", len(routes.paths)))
		}
		return nil, nil //nolint:nilnil // nil means "no such listener"; serve skips it.
	}

	if !isLoopback(address) {
		return nil, fmt.Errorf("httpServer.%s %q is not a loopback host:port", key, address)
	}

	listener, err := listen(context.Background(), "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on httpServer.%s %q: %w", key, address, err)
	}

	router := http.NewServeMux()
	routes.mountOn(router, cphttpserver.MiddlewareStack(cphttpserver.NewLoggingMiddleware(options.Logger)))

	return &sideServer{
		name:     name,
		server:   &http.Server{Handler: router, ReadHeaderTimeout: readHeaderTimeout},
		listener: listener,
	}, nil
}

func listenAdmin(options Options, listen listenFunc, routes *rpcRoutes) (*sideServer, error) {
	address, gate := options.Server.AdminBindAddress, options.AdminGate
	if address == "" {
		if !gate.empty() {
			options.Logger.Info("admin listener off, its gate unused")
		}
		return nil, nil //nolint:nilnil // nil means "no such listener"; serve skips it.
	}

	if !hostPort(address) {
		return nil, fmt.Errorf("httpServer.adminBindAddress %q is not a host:port", address)
	}
	if gate.empty() {
		return nil, fmt.Errorf(
			"httpServer.adminBindAddress %q is set and nothing gates it: the admin services have no authentication of their own",
			address,
		)
	}
	if slices.Contains(gate.Interceptors, nil) || slices.ContainsFunc(gate.Middlewares, func(m func(http.Handler) http.Handler) bool { return m == nil }) {
		return nil, errors.New("the admin gate holds a nil interceptor or middleware")
	}

	router := http.NewServeMux()
	middlewares := cphttpserver.MiddlewareStack(append(
		[]func(http.Handler) http.Handler{cphttpserver.NewLoggingMiddleware(options.Logger)}, gate.Middlewares...,
	)...)
	if err := routes.mountBehind(router, middlewares, gate.Interceptors...); err != nil {
		return nil, err
	}

	listener, err := listen(context.Background(), "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on httpServer.adminBindAddress %q: %w", address, err)
	}

	return &sideServer{
		name:     "admin",
		server:   &http.Server{Handler: router, ReadHeaderTimeout: readHeaderTimeout},
		listener: listener,
	}, nil
}

func serve(
	ctx context.Context,
	options Options,
	public net.Listener,
	router http.Handler,
	sides []*sideServer,
	drain context.CancelFunc,
	runners *runnerRegistry,
	closers *closerRegistry,
) error {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	server := &http.Server{
		Addr:      options.Server.BindAddress,
		Handler:   router,
		Protocols: protocols,

		// Only the header read is bounded: live streams are responses that stay open for hours.
		ReadHeaderTimeout: readHeaderTimeout,
	}

	running, stopRunning := context.WithCancel(ctx)
	defer stopRunning()

	started := startRunners(running, runners, options.Logger)

	options.Logger.Info("Listening", slog.String("address", server.Addr))

	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(public)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	for _, side := range sides {
		options.Logger.Info("Listening for "+side.name, slog.String("address", side.listener.Addr().String()))
		go func() {
			if err := side.server.Serve(side.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				options.Logger.Error(side.name+" server stopped", slog.Any("error", err))
			}
		}()
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-serveErr:
		for _, side := range sides {
			_ = side.server.Close()
		}
		stop(options, closers, stopRunning, started)
		if err != nil {
			return fmt.Errorf("failed to serve: %w", err)
		}
		return nil

	case sig := <-signals:
		options.Logger.Info("shutting down", slog.String("signal", sig.String()))

	case <-ctx.Done():
		options.Logger.Info("shutting down", slog.String("reason", "context cancelled"))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), options.ShutdownTimeout)
	defer cancel()

	// Before Shutdown, which would otherwise wait out its deadline on every open stream.
	drain()

	if err := server.Shutdown(shutdownCtx); err != nil {
		options.Logger.Error("failed to shut down the http server", slog.Any("error", err))
	}
	for _, side := range sides {
		if err := side.server.Shutdown(shutdownCtx); err != nil {
			options.Logger.Error("failed to shut down the "+side.name+" server", slog.Any("error", err))
		}
	}

	stop(options, closers, stopRunning, started)
	options.Logger.Info("shutdown complete")

	return nil
}

const readHeaderTimeout = 10 * time.Second

func startRunners(ctx context.Context, runners *runnerRegistry, logger *slog.Logger) *sync.WaitGroup {
	started := &sync.WaitGroup{}

	for _, runner := range runners.runners {
		started.Add(1)
		go func() {
			defer started.Done()
			runner.Run(ctx)
			logger.Debug("runner stopped", slog.String("runner", runner.Name()))
		}()
	}

	return started
}

func stop(options Options, closers *closerRegistry, stopRunning context.CancelFunc, started *sync.WaitGroup) {
	for _, closer := range closers.all() {
		if err := closer.close(); err != nil {
			options.Logger.Error("failed to close",
				slog.String("closer", closer.name),
				slog.Any("error", err),
			)
		}
	}

	stopRunning()
	started.Wait()
}
