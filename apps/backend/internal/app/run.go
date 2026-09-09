package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
)

const shutdownTimeout = 5 * time.Second

func (a *App) Run() error {
	a.logger.Info("Listening",
		lf.String("address", a.server.Addr),
	)

	runners := sync.WaitGroup{}
	for _, runner := range a.runners {
		runners.Add(1)
		go func() {
			defer runners.Done()
			runner()
		}()
	}

	serveErr := make(chan error, 1)
	go func() {
		err := a.server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-serveErr:
		a.stopRunners(&runners)
		if err != nil {
			return fmt.Errorf("failed to serve: %w", err)
		}
		return nil

	case sig := <-signalCh:
		a.logger.Info("shutting down", lf.String("signal", sig.String()))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("failed to shut down the http server", lf.Err(err))
	}

	a.stopRunners(&runners)
	a.logger.Info("shutdown complete")

	return nil
}

func (a *App) stopRunners(runners *sync.WaitGroup) {
	for _, shutdown := range a.shutdownFuncs {
		shutdown()
	}

	a.cancel()
	runners.Wait()
}
