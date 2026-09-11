package app

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/sequencer"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/primary/http/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/in_memory_country_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/in_memory_tile_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/x_publisher"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service/prom_click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
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

	a.configureBookkeeperIfEnabled(tilesStorage)

	clickLimiter := ratelimit.New(a.config.RateLimiter, xtime.ActualProvider{})
	a.runners = append(a.runners, func() { clickLimiter.Run(a.ctx) })

	// nil when boxes are switched off, which is what makes ClaimBonus answer
	// Unimplemented and leaves the stream carrying exactly what it did before.
	bonuses := a.configureBonusesIfEnabled()

	var booster planetv1controller.ClickBooster
	if bonuses != nil {
		booster = clickLimiter
	}

	clickService := planetv1controller.NewClickService(
		clickHandlerService,
		tilesChecker,
		tilesStorage,
		tilesStorage,
		a.config.HTTPServer.StreamHeartbeat,
		clickLimiter,
		bonuses,
		booster,
		xtime.ActualProvider{},
	)
	errorInterceptor := planetv1controller.NewErrorInterceptor(a.logger)

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

	// Innermost, after the throttle: a shadow-banned caller must keep hitting the same 429s, or never being throttled is the tell.
	antiBotInterceptor, err := a.configureAntiBot(tilesStorage)
	if err != nil {
		return err
	}
	if antiBotInterceptor != nil {
		interceptors = append(interceptors, antiBotInterceptor)
	}

	a.mountRPC(planetv1connect.NewClickServiceHandler(
		clickService,
		connect.WithInterceptors(interceptors...),
	))

	return nil
}

// configureBonusesIfEnabled returns a typed nil-free registry, or nil when boxes
// are off. Returning the interface rather than the struct matters: a nil
// *bonus.Registry in an interface is not a nil interface, and the stream tests
// for nil to decide whether to attend at all.
func (a *App) configureBonusesIfEnabled() planetv1controller.BonusRegistry {
	if !a.config.Bonus.Enabled {
		return nil
	}

	registry := bonus.New(a.config.Bonus, xtime.ActualProvider{})
	a.runners = append(a.runners, func() { registry.Run(a.ctx) })

	a.logger.Info("bonus boxes enabled",
		lf.Any("interval", a.config.Bonus.Interval),
		lf.Any("duration", a.config.Bonus.Duration),
	)

	return registry
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

// configureAntiBot assembles the watchdogs the file asks for, the jury that
// crosses them and the one shadow ban they all pass. It returns nil when
// nothing is enabled, which leaves the click chain exactly as it was.
func (a *App) configureAntiBot(owner planetv1controller.TileOwner) (connect.Interceptor, error) {
	config := a.config.AntiBot
	if !config.Enabled {
		return nil, nil
	}

	clock := xtime.ActualProvider{}

	onReaction, onFlag, err := planetv1controller.NewAntiBotReporter(a.logger, a.promRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot reporter: %w", err)
	}

	var (
		watchdogs []antibot.Watchdog
		names     []string
	)

	if config.Retaker.Enabled {
		watchdog := retaker.New(config.Retaker.Detector, clock, onReaction)
		a.runners = append(a.runners, func() { watchdog.Run(a.ctx) })
		watchdogs = append(watchdogs, watchdog)
		names = append(names, retaker.Name)
	}

	if config.Sequencer.Enabled {
		watchdog := sequencer.New(config.Sequencer.Detector, clock)
		a.runners = append(a.runners, func() { watchdog.Run(a.ctx) })
		watchdogs = append(watchdogs, watchdog)
		names = append(names, sequencer.Name)
	}

	if config.Metronome.Enabled {
		watchdog := metronome.New(config.Metronome.Detector, clock)
		a.runners = append(a.runners, func() { watchdog.Run(a.ctx) })
		watchdogs = append(watchdogs, watchdog)
		names = append(names, metronome.Name)
	}

	if len(watchdogs) == 0 {
		return nil, fmt.Errorf("antiBot is enabled with no watchdog turned on")
	}

	banner := shadowban.New(config.ShadowBan, clock)
	a.runners = append(a.runners, func() { banner.Run(a.ctx) })

	jury := antibot.NewJury(config.Jury, banner, clock, onFlag, watchdogs...)
	a.runners = append(a.runners, func() { jury.Run(a.ctx) })

	a.logger.Info("antibot enabled",
		lf.Any("watchdogs", names),
		lf.Int("minSuspects", config.Jury.MinSuspects),
		lf.Bool("enforce", banner.Enforcing()),
	)

	interceptor, err := planetv1controller.NewAntiBotInterceptor(jury, owner, clock, a.promRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot interceptor: %w", err)
	}

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
