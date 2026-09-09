package app

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/primary/http/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/in_memory_country_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/in_memory_tile_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/x_publisher"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service/prom_click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/wspublisher"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// countryChecker is the shared country list. Both contexts declare a port of
// this shape and neither owns the other's: the composition root hands the one
// implementation to both.
func (a *App) countryChecker() domain.CountryChecker {
	return in_memory_country_checker.New()
}

func (a *App) configureClicks(_ context.Context) error {
	a.configurePromRegistryIfNeeded()

	tilesChecker := in_memory_tile_checker.New(a.config.GameMap.MaxIndex)

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
		a.countryChecker(),
	)

	clickHandlerService, err := prom_click_handler_service.New(clickHandlerService, a.promRegistry)
	if err != nil {
		return fmt.Errorf("failed to create prometheus click handler service: %w", err)
	}

	// Bound to the app's lifetime, not to the startup context: cancelling it
	// closes the channel and stops the publisher during shutdown.
	updatesCh, err := tilesStorage.Subscribe(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to tile updates: %w", err)
	}

	publisher := wspublisher.New(
		updatesCh,
		planetv1controller.TileUpdateRoute,
		planetv1controller.EncodeTileUpdate,
		a.logger,
	)
	a.runners = append(a.runners, publisher.Run)
	a.mountWS(publisher.DeclareRoutes)

	a.configureBookkeeperIfEnabled(tilesStorage)

	clickService := planetv1controller.NewClickService(clickHandlerService, tilesChecker, tilesStorage)
	errorInterceptor := planetv1controller.NewErrorInterceptor(a.logger)

	clickLimiter := ratelimit.New(a.config.RateLimiter, xtime.ActualProvider{})
	a.runners = append(a.runners, func() { clickLimiter.Run(a.ctx) })

	vpnBlockInterceptor, err := a.configureVPNBlocklist()
	if err != nil {
		return err
	}

	// The error interceptor sits outside the other two so that everything the
	// handler chain answers goes through the same error mapping; both refusals
	// already carry their own code, which that mapping leaves alone.
	//
	// The blocklist sits outside the limiter: a refused address must not also
	// spend a token, or its next click would come back 429 and the web app
	// would show the throttle dialog rather than the VPN one.
	a.mountRPC(planetv1connect.NewClickServiceHandler(
		clickService,
		connect.WithInterceptors(
			errorInterceptor,
			vpnBlockInterceptor,
			planetv1controller.NewRateLimitInterceptor(clickLimiter),
		),
	))

	return nil
}

// configureVPNBlocklist parses the vendored VPN ranges and returns the
// interceptor that refuses clicks from them. A disabled config yields a nil
// *ipblock.Blocklist, which blocks nothing, so the interceptor stays in the
// chain either way and there is no second wiring path to keep in step.
func (a *App) configureVPNBlocklist() (connect.Interceptor, error) {
	blocklist, err := ipblock.New(a.config.VPNBlocklist)
	if err != nil {
		return nil, fmt.Errorf("failed to build vpn blocklist: %w", err)
	}

	if sizes := blocklist.Sizes(); len(sizes) > 0 {
		a.logger.Info("vpn blocklist enabled", lf.Any("ranges", sizes))
	}

	interceptor, err := planetv1controller.NewVPNBlockInterceptor(blocklist, a.promRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create vpn block interceptor: %w", err)
	}

	return interceptor, nil
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
