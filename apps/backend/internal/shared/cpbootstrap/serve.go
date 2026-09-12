package cpbootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

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

func serve(
	ctx context.Context,
	options Options,
	router http.Handler,
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

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-serveErr:
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

	if err := server.Shutdown(shutdownCtx); err != nil {
		options.Logger.Error("failed to shut down the http server", slog.Any("error", err))
	}

	stop(options, closers, stopRunning, started)
	options.Logger.Info("shutdown complete")

	return nil
}

func startRunners(ctx context.Context, runners *runnerRegistry, logger *slog.Logger) *sync.WaitGroup {
	started := &sync.WaitGroup{}

	for _, runner := range runners.runners {
		started.Add(1)
		go func() {
			defer started.Done()
			runner.run(ctx)
			logger.Debug("runner stopped", slog.String("runner", runner.name))
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
