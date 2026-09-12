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
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/get_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/map_density_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/geodesic_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/in_memory_tile_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/claim_bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/claim_bonus/prom_claim_bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/bonus_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/prom_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_budget"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/map_density"
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
	clock := cptime.SystemClock{}

	// First: nothing else here is worth starting if the map is not the one the frontend draws.
	if err := loadMapGeography(config.GameMap.MaxIndex, props); err != nil {
		return err
	}

	tilesChecker := in_memory_tile_checker.New(config.GameMap.MaxIndex)

	tilesStorage := memory_tile_storage.New(config.GameMap.MaxIndex, config.TilesStorage, props.Logger)
	props.Runners.Add("tiles-storage", tilesStorage.Run)

	limiter := cpratelimit.New(config.RateLimiter, clock)
	props.Runners.Add("click-limiter", limiter.Run)

	// nil when boxes are off, which leaves the feed and the click chain exactly
	// as they were and makes ClaimBonus answer Unimplemented.
	bonuses := newBonusRegistry(config.Bonus, clock, props)

	clickUseCase, err := clickChain(config, tilesChecker, tilesStorage, limiter, bonuses, props)
	if err != nil {
		return err
	}

	interceptors, err := edgeChain(config, props)
	if err != nil {
		return err
	}

	claimBonus, err := claimBonusUseCase(bonuses, limiter, clock, props)
	if err != nil {
		return err
	}

	// Each use case is handed only what it reads or writes, which is why the
	// storage appears three times here rather than once as a single object the
	// service holds: the map reader, the subscription and the tile writer are
	// three ports that happen to be served by one adapter.
	service := planetv1controller.ClickService{
		ClickHandler:      click_handler.New(clickUseCase),
		GetBudgetHandler:  get_budget_handler.New(get_budget.New(limiter)),
		MapDensityHandler: map_density_handler.New(map_density.New(tilesChecker)),
		GetMapHandler:     get_map_handler.New(get_map.New(tilesChecker, tilesStorage)),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events.New(tilesStorage, props.Server.StreamHeartbeat, bonusFeed(bonuses))),
		ClaimBonusHandler: claim_bonus_handler.New(claimBonus),
	}

	return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return planetv1connect.NewClickServiceHandler(service, options...)
	}, interceptors...)
}

// Unconditional, and fatal: a blob that disagrees with the frontend renumbers every tile, and the
// snapshot on disk is numbered the old way. See CLAUDE.md, "Map geography".
func loadMapGeography(maxIndex uint32, props cpbootstrap.Props) error {
	started := time.Now()

	geography, asset, err := geodesic_map.Load(maxIndex)
	if err != nil {
		return fmt.Errorf("failed to load the map geography: %w", err)
	}

	stats := geography.Stats()
	props.Logger.Info("map geography loaded",
		slog.String("asset", asset),
		slog.Int("tiles", int(stats.Tiles)),
		slog.Int("edges", int(stats.Edges)),
		slog.Any("degrees", stats.Degrees),
		slog.Any("took", time.Since(started).Round(time.Millisecond)),
	)

	return nil
}

// clickChain wraps the rule in the policies that guard it, innermost first:
// count it, judge it, then charge it. The throttle is outermost of the three so
// a shadow-banned caller keeps hitting the same 429s everyone else does — a
// caller that is never throttled again has been told it is banned.
func clickChain(
	config Config,
	tilesChecker *in_memory_tile_checker.Checker,
	tilesStorage *memory_tile_storage.Storage,
	limiter *cpratelimit.Limiter,
	bonuses *bonus.Registry,
	props cpbootstrap.Props,
) (click.IUseCase, error) {
	useCase, err := prom_click.New(
		click.New(tilesChecker, tilesStorage, cpcountries.New()),
		props.Metrics,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus click use case: %w", err)
	}

	guarded, err := wrapWithAntiBot(useCase, config.AntiBot, tilesStorage, props)
	if err != nil {
		return nil, err
	}

	// Inside the throttle: presence is what a caller actually managed to do,
	// not what they attempted.
	if bonuses != nil {
		guarded = bonus_click.New(guarded, bonuses)
	}

	return throttle_click.New(guarded, limiter), nil
}

// newBonusRegistry returns nil when boxes are off. A typed nil in an interface
// is not a nil interface, which is why the concrete type is returned here and
// the two helpers below do the widening.
func newBonusRegistry(config bonus.Config, clock cptime.Clock, props cpbootstrap.Props) *bonus.Registry {
	if !config.Enabled {
		return nil
	}

	registry := bonus.New(config, clock)
	props.Runners.Add("bonus-boxes", registry.Run)

	props.Logger.Info("bonus boxes enabled",
		slog.Any("minInterval", config.MinInterval),
		slog.Any("maxInterval", config.MaxInterval),
		slog.Any("duration", config.Duration),
	)

	return registry
}

func bonusFeed(registry *bonus.Registry) listen_for_events.BonusFeed {
	if registry == nil {
		return nil
	}

	return registry
}

// claimBonusUseCase also hands the registry its counters, which is why it takes
// the metrics registerer: offered against caught is the only way to see whether
// the pacing and the flight time are set anywhere near right.
func claimBonusUseCase(
	registry *bonus.Registry,
	limiter *cpratelimit.Limiter,
	clock cptime.Clock,
	props cpbootstrap.Props,
) (claim_bonus_handler.UseCase, error) {
	if registry == nil {
		return nil, nil //nolint:nilnil // nil means "boxes are off"; the handler answers Unimplemented.
	}

	useCase, counters, err := prom_claim_bonus.New(claim_bonus.New(registry, limiter, clock), props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create the bonus claim use case: %w", err)
	}

	registry.Observe(bonus.Report{
		Offered: counters.Offered.Inc,
		Lapsed:  counters.Lapsed.Inc,
	})

	return useCase, nil
}

// edgeChain builds the interceptors in the order they wrap the handler: the
// cache marks, then the two refusals that must not spend a token.
//
// The error net is not here. cpbootstrap wraps it around every service it
// mounts, so no module has to remember it and none can leave it out.
//
// The throttle and the shadow ban are no longer here. Both are decorators over
// the click use case, which puts them inside every interceptor by construction
// — so "a refused click must not also spend a token" is now a property of the
// shape rather than a rule about list order that a test has to pin.
func edgeChain(config Config, props cpbootstrap.Props) ([]connect.Interceptor, error) {
	vpnBlockInterceptor, err := newVPNBlockInterceptor(config.VPNBlocklist, props)
	if err != nil {
		return nil, err
	}

	interceptors := []connect.Interceptor{
		planetv1controller.NewCacheInterceptor(),
		vpnBlockInterceptor,
	}

	sessionInterceptor, err := newSessionInterceptor(config.Session, props)
	if err != nil {
		return nil, err
	}
	if sessionInterceptor != nil {
		interceptors = append(interceptors, sessionInterceptor)
	}

	return interceptors, nil
}

// newSessionInterceptor builds this context's own verifier from the same
// `session:` block the session context mints with — same secret, same MAC — so
// neither module has to hand the other an object. Nil when sessions are off.
func newSessionInterceptor(config cpsession.Config, props cpbootstrap.Props) (connect.Interceptor, error) {
	if !config.Enabled {
		//nolint:nilnil // nil means "sessions are off"; clickChain skips it.
		return nil, nil
	}

	verifier, err := cpsession.NewSigner(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build the click session verifier: %w", err)
	}

	interceptor, err := planetv1controller.NewSessionInterceptor(
		verifier,
		cptime.SystemClock{},
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
