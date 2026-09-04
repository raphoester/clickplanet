package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/primary/http/clicks_controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/primary/http/websocket_publisher"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/in_memory_country_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/in_memory_tile_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/redis_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/x_publisher"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service/prom_click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// tilesStorage is what cmd/api needs out of a tile storage: the domain port,
// plus the update stream the websocket publisher reads and the recent-updates
// window the bookkeeper reads. Both drivers satisfy it.
type tilesStorage interface {
	domain.TileStorage
	Subscribe(ctx context.Context) (<-chan domain.TileUpdate, error)
	PastUpdates(ctx context.Context, duration time.Duration, now time.Time) ([]domain.TileUpdate, error)
}

func (a *App) configureAppV2(ctx context.Context) (*ConfigureAppResponse, error) {
	a.configurePromRegistryIfNeeded()
	a.configureHTTPFormatsIfNeeded()

	tilesChecker := in_memory_tile_checker.New(a.config.GameMap.MaxIndex)
	countryChecker := in_memory_country_checker.New()

	storage, err := a.buildTilesStorage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build tiles storage: %w", err)
	}

	var clickHandlerService click_handler_service.IService = click_handler_service.New(
		tilesChecker,
		storage,
		countryChecker,
	)

	clickHandlerService, err = prom_click_handler_service.New(clickHandlerService, a.promRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus click handler service: %w", err)
	}

	// Bound to the app's lifetime, not to the startup context: cancelling it
	// closes the channel and stops the publisher during shutdown.
	updatesCh, err := storage.Subscribe(a.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to tile updates: %w", err)
	}

	publisher := websocket_publisher.New(updatesCh, a.answerer)
	a.runners = append(a.runners, publisher.Run)

	a.configureBookkeeperIfEnabled(storage)

	controller := clicks_controller.New(
		clickHandlerService,
		tilesChecker,
		storage,
		a.answerer,
		a.reader,
	)

	return &ConfigureAppResponse{
		declareWSRoutes:  publisher.DeclareRoutes,
		declareRPCRoutes: controller.DeclareRoutes,
	}, nil
}

// buildTilesStorage picks the tile storage implementation from the config.
// Redis is only dialled when the redis driver is selected, so the default
// deployment is a single self-contained container.
func (a *App) buildTilesStorage(ctx context.Context) (tilesStorage, error) {
	driver := strings.ToLower(strings.TrimSpace(a.config.TilesStorage.Driver))

	switch driver {
	case "", DriverMemory:
		a.logger.Info("using the in-process tile storage", lf.String("driver", DriverMemory))

		storage := memory_tile_storage.New(
			a.config.GameMap.MaxIndex,
			a.config.TilesStorage.Memory,
			xtime.ActualProvider{},
			a.logger,
		)

		// Owns the snapshot loop: periodic flushes plus a final one when the
		// app context is cancelled.
		a.runners = append(a.runners, func() { storage.Run(a.ctx) })

		return storage, nil

	case DriverRedis:
		a.logger.Info("using the redis tile storage", lf.String("driver", DriverRedis))

		if err := a.configureRedisClientIfNeeded(ctx); err != nil {
			return nil, fmt.Errorf("failed to configure redis client: %w", err)
		}

		return redis_tile_storage.New(a.redisClient, a.config.TilesStorage.Redis), nil

	default:
		return nil, fmt.Errorf(
			"unknown tiles storage driver %q, expected %q or %q",
			driver, DriverMemory, DriverRedis,
		)
	}
}

// configureBookkeeperIfEnabled runs the X reporting job in-process. It used to
// be cmd/bookkeeper, but the memory driver's recent updates only exist inside
// the API process, so a separate process has nothing to read.
func (a *App) configureBookkeeperIfEnabled(storage tilesStorage) {
	if !a.config.Bookkeeper.Enabled {
		return
	}

	a.logger.Info("bookkeeper enabled", lf.Any("interval", a.config.Bookkeeper.Runner.Interval))

	bookkeeper := runner.New(
		a.config.Bookkeeper.Runner,
		x_publisher.New(),
		storage,
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
