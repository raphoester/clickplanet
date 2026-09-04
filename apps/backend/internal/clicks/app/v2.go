package app

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/primary/http/clicks_controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/primary/http/websocket_publisher"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/in_memory_country_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/in_memory_tile_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/x_publisher"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service/prom_click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

func (a *App) configureAppV2(_ context.Context) (*ConfigureAppResponse, error) {
	a.configurePromRegistryIfNeeded()
	a.configureHTTPFormatsIfNeeded()

	tilesChecker := in_memory_tile_checker.New(a.config.GameMap.MaxIndex)
	countryChecker := in_memory_country_checker.New()

	tilesStorage := memory_tile_storage.New(
		a.config.GameMap.MaxIndex,
		a.config.TilesStorage,
		xtime.ActualProvider{},
		a.logger,
	)

	// Owns the snapshot loop: periodic flushes plus a final one when the app
	// context is cancelled.
	a.runners = append(a.runners, func() { tilesStorage.Run(a.ctx) })

	var clickHandlerService click_handler_service.IService = click_handler_service.New(
		tilesChecker,
		tilesStorage,
		countryChecker,
	)

	clickHandlerService, err := prom_click_handler_service.New(clickHandlerService, a.promRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus click handler service: %w", err)
	}

	// Bound to the app's lifetime, not to the startup context: cancelling it
	// closes the channel and stops the publisher during shutdown.
	updatesCh, err := tilesStorage.Subscribe(a.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to tile updates: %w", err)
	}

	publisher := websocket_publisher.New(updatesCh, a.answerer)
	a.runners = append(a.runners, publisher.Run)

	a.configureBookkeeperIfEnabled(tilesStorage)

	controller := clicks_controller.New(
		clickHandlerService,
		tilesChecker,
		tilesStorage,
		a.answerer,
		a.reader,
	)

	return &ConfigureAppResponse{
		declareWSRoutes:  publisher.DeclareRoutes,
		declareRPCRoutes: controller.DeclareRoutes,
	}, nil
}

// configureBookkeeperIfEnabled runs the X reporting job in-process. It used to
// be cmd/bookkeeper, but the tile storage's recent updates only exist inside
// this process, so a separate process has nothing to read.
func (a *App) configureBookkeeperIfEnabled(tilesStorage *memory_tile_storage.Storage) {
	if !a.config.Bookkeeper.Enabled {
		return
	}

	a.logger.Info("bookkeeper enabled", lf.Any("interval", a.config.Bookkeeper.Runner.Interval))

	bookkeeper := runner.New(
		a.config.Bookkeeper.Runner,
		x_publisher.New(),
		tilesStorage,
		xtime.ActualProvider{},
		a.logger,
	)

	a.runners = append(a.runners, bookkeeper.Run)
	a.shutdownFuncs = append(a.shutdownFuncs, func() {
		if err := bookkeeper.GracefulShutdown(); err != nil {
			a.logger.Error("failed to shut down the bookkeeper", lf.Err(err))
		}
	})
}
