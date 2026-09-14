package admin_server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 5 * time.Second
)

// Listen binds up front, so a taken address refuses the boot.
func Listen(config Config) (net.Listener, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", config.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to listen on admin.bindAddress %q: %w", config.Address(), err)
	}

	return listener, nil
}

func Serve(ctx context.Context, listener net.Listener, handler http.Handler, logger *slog.Logger) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shut down the admin server", slog.String("error", err.Error()))
		}
	}()

	logger.Info("admin server listening", slog.String("address", listener.Addr().String()))

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("admin server stopped", slog.String("error", err.Error()))
	}

	<-done
}
