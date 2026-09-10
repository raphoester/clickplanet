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
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/clickbudget"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/wspublisher"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

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

	// Error mapping outermost, then the two refusals that must not spend a
	// token, then the throttle. A click refused for its address or its session
	// coming back 429 on the next attempt would send the client to the wrong
	// dialog entirely.
	interceptors := []connect.Interceptor{errorInterceptor, vpnBlockInterceptor}

	sessionInterceptor, err := a.configureClickSessions()
	if err != nil {
		return err
	}
	if sessionInterceptor != nil {
		interceptors = append(interceptors, sessionInterceptor)
	}

	interceptors = append(interceptors, planetv1controller.NewRateLimitInterceptor(clickLimiter))

	// Last, and after the throttle: a click the throttle refused never reached
	// the map, so charging a budget for it would bill the caller for nothing.
	budgetInterceptor, err := a.configureClickBudget()
	if err != nil {
		return err
	}
	if budgetInterceptor != nil {
		interceptors = append(interceptors, budgetInterceptor)
	}

	a.mountRPC(planetv1connect.NewClickServiceHandler(
		clickService,
		connect.WithInterceptors(interceptors...),
	))

	return nil
}

// configureClickSessions returns nil when sessions are disabled, which leaves
// the click chain exactly as it was.
func (a *App) configureClickSessions() (connect.Interceptor, error) {
	if a.sessionSigner == nil {
		return nil, nil
	}

	interceptor, err := planetv1controller.NewSessionInterceptor(
		a.sessionSigner,
		xtime.ActualProvider{},
		a.config.Session.Enforce,
		a.promRegistry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the click session interceptor: %w", err)
	}

	return interceptor, nil
}

// configureClickBudget returns nil unless sessions are on and a budget is
// configured. Without a session there is no ID to charge, so a budget would
// have nothing to key on and would silently pass every click.
func (a *App) configureClickBudget() (connect.Interceptor, error) {
	if a.sessionSigner == nil {
		return nil, nil
	}

	// Retention comes from the signer rather than from the config: a tally
	// forgotten while its token is still valid would hand that token a second
	// budget, and taking the TTL from the one thing that enforces it is what
	// makes that impossible to misconfigure.
	budget := clickbudget.New(
		a.config.Session.ClickBudget,
		a.sessionSigner.TTL(),
		xtime.ActualProvider{},
	)
	if budget == nil {
		return nil, nil
	}

	a.runners = append(a.runners, func() { budget.Run(a.ctx) })

	interceptor, err := planetv1controller.NewClickBudgetInterceptor(budget, a.promRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create the click budget interceptor: %w", err)
	}

	a.logger.Info("click budget enabled", lf.Int("clicks", a.config.Session.ClickBudget.Clicks))

	return interceptor, nil
}

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
