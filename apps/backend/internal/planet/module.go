// Package planet wires the tile game: the map, the click chain and the live
// stream. It is named for its proto package, planet.v1, the way chat and session
// are for theirs — Click is one procedure on the service, not the whole of it.
//
// It is the one module that is never off — a process without it is not this game
// — so it has no Enabled switch, only the ones inside it.
package planet

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/in_memory_tile_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain/click_handler_service/prom_click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "planet"

// NewModule is always enabled: a process without the tile game is not this game.
func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(_ context.Context, props cpbootstrap.Props) error {
			return build(config, props)
		},
	}
}

func build(config Config, props cpbootstrap.Props) error {
	clock := cptime.ActualProvider{}

	tilesChecker := in_memory_tile_checker.New(config.GameMap.MaxIndex)

	tilesStorage := memory_tile_storage.New(config.GameMap.MaxIndex, config.TilesStorage, props.Logger)
	props.Runners.Add("tiles-storage", tilesStorage.Run)

	var handler click_handler_service.IService = click_handler_service.New(tilesChecker, tilesStorage, cpcountries.New())
	handler, err := prom_click_handler_service.New(handler, props.Metrics)
	if err != nil {
		return fmt.Errorf("failed to create prometheus click handler service: %w", err)
	}

	clickLimiter := cpratelimit.New(config.RateLimiter, clock)
	props.Runners.Add("click-limiter", clickLimiter.Run)

	interceptors, err := clickChain(config, tilesStorage, clickLimiter, props)
	if err != nil {
		return err
	}

	return props.RPC.Mount(planetv1connect.NewClickServiceHandler(
		planetv1controller.NewClickService(
			handler,
			tilesChecker,
			tilesStorage,
			tilesStorage,
			props.Server.StreamHeartbeat,
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
	owner planetv1controller.TileOwner,
	clickLimiter *cpratelimit.Limiter,
	props cpbootstrap.Props,
) ([]connect.Interceptor, error) {
	vpnBlockInterceptor, err := newVPNBlockInterceptor(config.VPNBlocklist, props)
	if err != nil {
		return nil, err
	}

	interceptors := []connect.Interceptor{
		planetv1controller.NewErrorInterceptor(props.Logger),
		vpnBlockInterceptor,
	}

	sessionInterceptor, err := newSessionInterceptor(config.Session, props)
	if err != nil {
		return nil, err
	}
	if sessionInterceptor != nil {
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

// newSessionInterceptor builds this context's own verifier from the same
// `session:` block the session context mints with — same secret, same MAC — so
// neither module has to hand the other an object. Nil when sessions are off.
func newSessionInterceptor(config cpsession.Config, props cpbootstrap.Props) (connect.Interceptor, error) {
	if !config.Enabled {
		return nil, nil
	}

	verifier, err := cpsession.NewSigner(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build the click session verifier: %w", err)
	}

	interceptor, err := planetv1controller.NewSessionInterceptor(
		verifier,
		cptime.ActualProvider{},
		config.Enforce,
		props.Metrics,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the click session interceptor: %w", err)
	}

	return interceptor, nil
}

func newVPNBlockInterceptor(config cpipblock.Config, props cpbootstrap.Props) (connect.Interceptor, error) {
	blocklist, err := cpipblock.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build vpn blocklist: %w", err)
	}

	if sizes := blocklist.Sizes(); len(sizes) > 0 {
		props.Logger.Info("vpn blocklist enabled", slog.Any("ranges", sizes))
	}

	interceptor, err := planetv1controller.NewVPNBlockInterceptor(blocklist, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create vpn block interceptor: %w", err)
	}

	return interceptor, nil
}
