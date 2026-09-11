// Package clicks wires the tile game: the map, the click chain and the live
// stream. It is the one module that is never off — a process without it is not
// this game — so it has no Enabled switch, only the ones inside it.
package clicks

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/primary/http/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/in_memory_tile_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/x_publisher"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service/prom_click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/bootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/session"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

const moduleName = "clicks"

// Deps is what the composition root owns rather than this context.
//
// Signer is the worked example: sessions mint with it and clicks verifies with
// it, so it belongs to neither and is handed to both. Nil means sessions are
// off, which leaves the click chain exactly as it was before they existed.
type Deps struct {
	Countries domain.CountryChecker

	Signer *session.Signer

	// Off counts what enforcing would refuse without refusing it. Read here and
	// not from a config of ours, because it is the session context's rollout
	// switch and this one only ever checks a signature.
	EnforceSessions bool

	// Must stay well under the proxy's idle cut: Cloudflare answers 524 at ~125s.
	StreamHeartbeat time.Duration
}

func NewModule(config Config, deps Deps) bootstrap.Module {
	return bootstrap.Module{
		Name: moduleName,
		DiSequence: func(_ context.Context, props bootstrap.Props) error {
			return build(config, deps, props)
		},
	}
}

func build(config Config, deps Deps, props bootstrap.Props) error {
	clock := xtime.ActualProvider{}

	tilesChecker := in_memory_tile_checker.New(config.GameMap.MaxIndex)

	tilesStorage := memory_tile_storage.New(config.GameMap.MaxIndex, config.TilesStorage, clock, props.Logger)
	props.Runners.Add("tiles-storage", tilesStorage.Run)

	var handler click_handler_service.IService = click_handler_service.New(tilesChecker, tilesStorage, deps.Countries)
	handler, err := prom_click_handler_service.New(handler, props.Metrics)
	if err != nil {
		return fmt.Errorf("failed to create prometheus click handler service: %w", err)
	}

	if err := addBookkeeper(config.Bookkeeper, tilesStorage, props); err != nil {
		return err
	}

	clickLimiter := ratelimit.New(config.RateLimiter, clock)
	props.Runners.Add("click-limiter", clickLimiter.Run)

	interceptors, err := clickChain(config, deps, tilesStorage, clickLimiter, props)
	if err != nil {
		return err
	}

	return props.RPC.Mount(planetv1connect.NewClickServiceHandler(
		planetv1controller.NewClickService(
			handler,
			tilesChecker,
			tilesStorage,
			tilesStorage,
			deps.StreamHeartbeat,
			clickLimiter,
		),
		connect.WithInterceptors(interceptors...),
	))
}

// clickChain builds the interceptors in the order they wrap the handler.
//
// Error mapping outermost, then the two refusals that must not spend a token,
// then the throttle, then the shadow ban. A click refused for its address or
// its session coming back 429 on the next attempt would send the client to the
// wrong dialog entirely; a shadow-banned caller that is never throttled again
// has been told it is banned.
func clickChain(
	config Config,
	deps Deps,
	owner planetv1controller.TileOwner,
	clickLimiter *ratelimit.Limiter,
	props bootstrap.Props,
) ([]connect.Interceptor, error) {
	vpnBlockInterceptor, err := newVPNBlockInterceptor(config.VPNBlocklist, props)
	if err != nil {
		return nil, err
	}

	interceptors := []connect.Interceptor{
		planetv1controller.NewErrorInterceptor(props.Logger),
		vpnBlockInterceptor,
	}

	if deps.Signer != nil {
		sessionInterceptor, err := planetv1controller.NewSessionInterceptor(
			deps.Signer,
			xtime.ActualProvider{},
			deps.EnforceSessions,
			props.Metrics,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create the click session interceptor: %w", err)
		}
		interceptors = append(interceptors, sessionInterceptor)
	}

	interceptors = append(interceptors, planetv1controller.NewRateLimitInterceptor(clickLimiter))

	antiBotInterceptor, err := newAntiBotInterceptor(config.AntiBot, owner, props)
	if err != nil {
		return nil, err
	}
	if antiBotInterceptor != nil {
		interceptors = append(interceptors, antiBotInterceptor)
	}

	return interceptors, nil
}

func newVPNBlockInterceptor(config ipblock.Config, props bootstrap.Props) (connect.Interceptor, error) {
	blocklist, err := ipblock.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build vpn blocklist: %w", err)
	}

	if sizes := blocklist.Sizes(); len(sizes) > 0 {
		props.Logger.Info("vpn blocklist enabled", lf.Any("ranges", sizes))
	}

	interceptor, err := planetv1controller.NewVPNBlockInterceptor(blocklist, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create vpn block interceptor: %w", err)
	}

	return interceptor, nil
}

func addBookkeeper(config BookkeeperConfig, storage *memory_tile_storage.Storage, props bootstrap.Props) error {
	if !config.Enabled {
		return nil
	}

	props.Logger.Info("bookkeeper enabled", lf.Any("interval", config.Runner.Interval))

	bookkeeper := runner.New(config.Runner, x_publisher.New(), storage, xtime.ActualProvider{}, props.Logger)

	// The bookkeeper stops on its own channel rather than on a context, so the
	// cleanup is what ends it and the runner is only waited on afterwards.
	props.Runners.Add("bookkeeper", func(context.Context) { bookkeeper.Run() })
	props.Closers.Add("bookkeeper", bookkeeper.GracefulShutdown)

	return nil
}
