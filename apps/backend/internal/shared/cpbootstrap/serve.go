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
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpprom"
)

// mountMetrics puts the scrape endpoint on the same router as the RPC routes.
// It carries the logging middleware alone: the CORS and IP headers are for
// callers of the API, and nothing browses this.
func mountMetrics(router *http.ServeMux, metrics *prometheus.Registry, logger *slog.Logger) {
	metricsRouter := http.NewServeMux()
	metricsRouter.HandleFunc("GET /", cpprom.HandlerForRegistry(metrics).ServeHTTP)

	router.Handle("/metrics", cphttpserver.MiddlewareStack(
		cphttpserver.NewLoggingMiddleware(logger),
	)(metricsRouter))
}

type loopbackServer struct {
	name     string
	server   *http.Server
	listener net.Listener
}

var errNoLoopbackAddress = errors.New("no address configured")

// listenLoopback binds before anything is served, so a taken address refuses the boot. Empty is errNoLoopbackAddress.
func listenLoopback(options Options, name, key, address string, routes *rpcRoutes) (*loopbackServer, error) {
	if address == "" {
		return nil, fmt.Errorf("httpServer.%s: %w", key, errNoLoopbackAddress)
	}

	if !isLoopback(address) {
		return nil, fmt.Errorf("httpServer.%s %q is not a loopback host:port", key, address)
	}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on httpServer.%s %q: %w", key, address, err)
	}

	router := http.NewServeMux()
	routes.mountOn(router, cphttpserver.MiddlewareStack(cphttpserver.NewLoggingMiddleware(options.Logger)))

	return &loopbackServer{
		name:     name,
		server:   &http.Server{Handler: router, ReadHeaderTimeout: readHeaderTimeout},
		listener: listener,
	}, nil
}

func serve(
	ctx context.Context,
	options Options,
	router http.Handler,
	loopbacks []*loopbackServer,
	drain context.CancelFunc,
	runners *runnerRegistry,
	closers *closerRegistry,
) error {
	// The generated handlers speak gRPC and gRPC-Web as well as Connect, and
	// those need HTTP/2; browsers reach the same routes over HTTP/1.1.
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	server := &http.Server{
		Addr:      options.Server.BindAddress,
		Handler:   router,
		Protocols: protocols,

		// A connection that opens and then dribbles its headers holds a goroutine
		// open for as long as it likes; enough of them is the whole attack. Only
		// the header read is bounded — the body and the response are not, because
		// the live streams are responses that stay open for hours by design.
		ReadHeaderTimeout: readHeaderTimeout,
	}

	running, stopRunning := context.WithCancel(ctx)
	defer stopRunning()

	started := startRunners(running, runners, options.Logger)

	options.Logger.Info("Listening", slog.String("address", server.Addr))

	serveErr := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	for _, loopback := range loopbacks {
		options.Logger.Info("Listening for "+loopback.name, slog.String("address", loopback.listener.Addr().String()))
		go func() {
			if err := loopback.server.Serve(loopback.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				options.Logger.Error(loopback.name+" server stopped", slog.Any("error", err))
			}
		}()
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-serveErr:
		for _, loopback := range loopbacks {
			_ = loopback.server.Close()
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

	// Before Shutdown, which waits for every connection to go idle and would
	// otherwise wait out its deadline on the first stream still open. Unary
	// calls in flight are left to finish: only the streams read this.
	drain()

	if err := server.Shutdown(shutdownCtx); err != nil {
		options.Logger.Error("failed to shut down the http server", slog.Any("error", err))
	}
	// After the public server: a public call still in flight may be waiting on an internal one.
	for _, loopback := range loopbacks {
		if err := loopback.server.Shutdown(shutdownCtx); err != nil {
			options.Logger.Error("failed to shut down the "+loopback.name+" server", slog.Any("error", err))
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

// stop runs the cleanups before cancelling the runners, because a cleanup is
// how a runner with no context of its own is asked to return: cancelling first
// would leave nothing to ask.
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
