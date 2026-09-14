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

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/ban_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/drop_bomb_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/find_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/get_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/map_density_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/reassign_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/revert_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/geodesic_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/in_memory_tile_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/pacing"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/toll"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/ban_player"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/ban_player/audit_ban"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/claim_bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/claim_bonus/prom_claim_bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/antibot_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/bonus_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/enclose_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/enclose_click/prom_enclose"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/prom_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/spread_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/drop_bomb/antibot_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/drop_bomb/prom_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/find_players"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_budget"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/map_density"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country/audit_reassign"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/revert_player"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/revert_player/audit_revert"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "planet"

// NewModule is always enabled: a process without the tile game is not this game.
//
// The whole DI sequence is the one function below, top to bottom, in the order
// things are built and registered.
func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(_ context.Context, props cpbootstrap.Props) error {
			clock := cptime.SystemClock{}
			countries := cpcountries.New()

			// ---- Map geography ----

			// First: nothing else here is worth starting if the map is not the one the frontend draws.
			// Unconditional, and fatal: a blob that disagrees with the frontend renumbers every tile, and the
			// snapshot on disk is numbered the old way. See CLAUDE.md, "Map geography".
			gameMap := geodesic_map.New(config.GameMap.MaxIndex, props.Logger)

			geography, err := gameMap.LoadGeography()
			if err != nil {
				return fmt.Errorf("failed to load the map geography: %w", err)
			}

			// Unconditional, and fatal, for the same reason: borders for another map name the wrong ground.
			borders, err := gameMap.LoadBorders()
			if err != nil {
				return fmt.Errorf("failed to load the map borders: %w", err)
			}

			// ---- Storage, ledger, limiter, toll ----

			tilesChecker := in_memory_tile_checker.New(config.GameMap.MaxIndex)

			tilesStorage := memory_tile_storage.New(config.GameMap.MaxIndex, config.TilesStorage, props.Logger)
			tilesStorage.LoadSnapshot()
			props.Runners.Add(tilesStorage)

			takings := ledger.New(config.Ledger, clock)
			props.Runners.Add(takings)

			limiter := cpratelimit.New("click-limiter", config.RateLimiter, clock)
			props.Runners.Add(limiter)

			pricer := toll.New(config.Toll, tilesStorage)

			// writer is the storage as the click chain writes it, so every tile it takes lands in the ledger.
			writer := ledger.Recording{Tiles: tilesStorage, Ledger: takings}

			// ---- Bonus boxes ----

			bonuses := bonus.New(config.Bonus, clock)
			props.Runners.Add(bonuses)

			spreads := bonus.NewSpreads(clock)
			bombs := bonus.NewBombs(clock)
			enclosures := bonus.NewEnclosures(clock)

			// A bomb is sized off the map itself, so the ring a client draws is the width of what it clears.
			spacing := geography.Spacing()
			bombRules := drop_bomb.Rules{
				Radius: config.Bonus.Rings() * spacing,
				// Within a tile of the nearest tile is land; further out is the sea.
				Reach: spacing,
			}

			// ---- Click chain ----

			// The rule is wrapped in the policies that guard it, innermost first:
			// spread or enclose it, count it, judge it, then charge it. The throttle is outermost so
			// a shadow-banned caller keeps hitting the same 429s everyone else does — a
			// caller that is never throttled again has been told it is banned.

			// Right against the rule, inside the shadow ban: a dropped click never
			// reaches the rule, so it spreads and encloses nothing either. It is counted
			// as one click however many tiles it took.
			var clickUseCase click.IUseCase = click.New(tilesChecker, writer, countries)
			clickUseCase = spread_click.New(clickUseCase, spreads, geography, writer, bonuses)

			clickUseCase = enclose_click.New(clickUseCase, enclosures,
				enclose_click.NewTerrain(geography, tilesStorage),
				enclose_click.NewAnnexer(writer, prom_enclose.New(bonuses, props.Metrics)))

			clickUseCase = prom_click.New(clickUseCase, props.Metrics)

			// The shadow ban. The metric names and the words of the ban line are the
			// observer's; the guard measures and judges.
			guard, err := antibot.New(config.AntiBot, clock, antibot_click.NewObserver(props.Logger, props.Metrics))
			if err != nil {
				return fmt.Errorf("failed to build the antibot guard: %w", err)
			}

			// With the antibot off the guard drops nothing: the click passes, BanPlayer
			// refuses, FindPlayers says nothing of bans and a bomb is never a dud.
			guard.LoadBans()
			props.Runners.Add(guard)

			clickUseCase = antibot_click.New(clickUseCase, guard, tilesStorage, clock, props.Metrics)

			// Inside the throttle: presence is what a caller actually managed to do,
			// not what they attempted.
			clickUseCase = bonus_click.New(clickUseCase, bonuses)

			clickUseCase = throttle_click.New(clickUseCase, limiter, pricer)

			// ---- Admin service ----

			// A quarter of a subscriber's buffer per batch leaves room for the clicks still arriving.
			adminBatch := config.TilesStorage.SubscriberBuffer / 4
			if adminBatch <= 0 {
				adminBatch = 256
			}
			pace := pacing.Pacing{Batch: adminBatch, Pause: 50 * time.Millisecond}

			adminService := planetv1controller.AdminService{
				ReassignCountryHandler: reassign_country_handler.New(audit_reassign.New(
					reassign_country.New(tilesStorage, countries, pace), props.Logger)),
				FindPlayersHandler: find_players_handler.New(
					find_players.New(takings, tilesStorage, borders, guard, countries)),
				BanPlayerHandler: ban_player_handler.New(audit_ban.New(ban_player.New(guard), props.Logger)),
				RevertPlayerHandler: revert_player_handler.New(
					audit_revert.New(revert_player.New(takings, tilesStorage, pace), props.Logger)),
			}

			if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return planetv1connect.NewAdminServiceHandler(adminService, options...)
			}); err != nil {
				return err
			}

			// ---- Edge interceptors ----

			// In the order they wrap the handler: the cache marks, then the two refusals
			// that must not spend a token.
			//
			// The error net is not here. cpbootstrap wraps it around every service it
			// mounts, so no module has to remember it and none can leave it out.
			//
			// The throttle and the shadow ban are not here either. Both are decorators
			// over the click use case, which puts them inside every interceptor by
			// construction — so "a refused click must not also spend a token" is a
			// property of the shape rather than a rule about list order.
			blocklist := cpipblock.New(config.VPNBlocklist)
			if err := blocklist.Load(); err != nil {
				return fmt.Errorf("failed to load the vpn blocklist: %w", err)
			}

			if sizes := blocklist.Sizes(); len(sizes) > 0 {
				props.Logger.Info("vpn blocklist enabled", slog.Any("ranges", sizes))
			}

			interceptors := []connect.Interceptor{
				planetv1controller.NewCacheInterceptor(),
				planetv1controller.NewVPNBlockInterceptor(blocklist, props.Metrics),
			}

			// This context builds its own verifier from the same `session:` block the
			// session context mints with — same secret, same MAC — so neither module
			// has to hand the other an object. Skipped when sessions are off.
			if config.Session.Enabled {
				verifier, err := cpsession.NewSigner(config.Session)
				if err != nil {
					return fmt.Errorf("failed to build the click session verifier: %w", err)
				}

				interceptors = append(interceptors, planetv1controller.NewSessionInterceptor(
					verifier,
					clock,
					config.Session.Enforce,
					props.Metrics))
			}

			// ---- Bonus use cases ----

			// The claim also hands the registry its counters: offered against caught
			// is the only way to see whether the pacing and the flight time are set
			// anywhere near right.
			claimBonus, counters := prom_claim_bonus.New(
				claim_bonus.New(bonuses, limiter, pricer, spreads, bombs, bombRules.Radius, enclosures, clock),
				props.Metrics)

			bonuses.Observe(bonus.Report{
				Offered: counters.Offered.Inc,
				Lapsed:  counters.Lapsed.Inc,
			})

			dropped := prom_drop_bomb.New(
				drop_bomb.New(bombs, bonuses, geography, tilesStorage, countries, bombRules), props.Metrics)

			// Outside the count, so it can tell the counter a drop was a dud.
			dropBomb := antibot_drop_bomb.New(dropped, guard)

			// ---- Click service ----

			// Each use case is handed only what it reads or writes, which is why the
			// storage appears three times here rather than once as a single object the
			// service holds: the map reader, the subscription and the tile writer are
			// three ports that happen to be served by one adapter.
			service := planetv1controller.ClickService{
				ClickHandler:      click_handler.New(clickUseCase),
				GetBudgetHandler:  get_budget_handler.New(get_budget.New(limiter, pricer)),
				MapDensityHandler: map_density_handler.New(map_density.New(tilesChecker)),
				GetMapHandler:     get_map_handler.New(get_map.New(tilesChecker, tilesStorage)),
				ListenForEventsHandler: listen_for_events_handler.New(
					listen_for_events.New(tilesStorage, props.Server.StreamHeartbeat, bonuses)),
				ClaimBonusHandler: claim_bonus_handler.New(claimBonus),
				DropBombHandler:   drop_bomb_handler.New(dropBomb),
			}

			return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return planetv1connect.NewClickServiceHandler(service, options...)
			}, interceptors...)
		},
	}
}
