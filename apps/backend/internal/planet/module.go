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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase/prom_claim_bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase/antibot_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase/prom_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_geodesic_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/postgres_tile_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/antibot_attempt_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/antibot_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/bonus_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click/prom_enclose"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/prom_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/spread_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_budget_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase/antibot_get_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase/antibot_listen_for_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/map_density_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase/audit_paint_random"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase/audit_reassign"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/ban_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/ban_player_usecase/audit_ban"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/find_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/inspect_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase/audit_revert"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/top_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/ban_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/drop_bomb_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/find_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/inspect_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/map_density_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/paint_random_tiles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/reassign_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/revert_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/top_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
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
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			clock := cptime.SystemClock{}
			countries := cpcountries.New()

			// ---- Map geography ----

			// First: nothing else here is worth starting if the map is not the one the frontend draws.
			// Unconditional, and fatal: a blob that disagrees with the frontend renumbers every tile, and the
			// tiles in postgres are numbered the old way. See CLAUDE.md, "Map geography".
			gameMap := embedded_geodesic_map.New(config.GameMap.MaxIndex, props.Logger)

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

			tilesChecker := clicks.NewBoard(config.GameMap.MaxIndex)

			db := cppg.New(config.Database)
			if err := db.ConnectCtx(ctx); err != nil {
				return fmt.Errorf("failed to connect the planet to postgres: %w", err)
			}
			if err := db.Migrate(ctx, migrations.FS); err != nil {
				_ = db.Close()
				return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
			}

			tilesStorage := inmemory_tile_storage.New(
				config.GameMap.MaxIndex, config.TilesStorage, postgres_tile_store.New(db), props.Logger)
			if err := tilesStorage.Load(ctx); err != nil {
				_ = db.Close()
				return fmt.Errorf("failed to load the tile map: %w", err)
			}

			takings := inmemory_ledger_storage.New(config.LedgerStorage, postgres_ledger_store.New(db), props.Logger)
			if err := takings.Load(ctx); err != nil {
				_ = db.Close()
				return fmt.Errorf("failed to load the ledger: %w", err)
			}

			// The pool closes after both runners' last flush, not as a closer: closers run first.
			props.Runners.Add(cppg.CloseAfter(db, props.Logger, tilesStorage, takings))
			props.Runners.Add(ledger.NewRetention(config.Ledger, takings, clock))

			limiter := cpratelimit.New("click-limiter", config.RateLimiter, clock)
			props.Runners.Add(limiter)

			pricer := clicks.NewToll(config.Toll, tilesStorage)

			// writer is the storage as the click chain writes it, so every tile it takes lands in the ledger.
			writer := ledger.NewRecording(tilesStorage, takings, clock)

			// ---- Bonus boxes ----

			registry := bonuses.New(config.Bonus, clock)
			props.Runners.Add(registry)

			spreads := bonuses.NewSpreads(clock)
			bombs := bonuses.NewBombs(clock)
			enclosures := bonuses.NewEnclosures(clock)

			bombRules := bonuses.NewBombRules(config.Bonus.Bomb, geography.Spacing())

			// ---- Click chain ----

			// The rule is wrapped in the policies that guard it, innermost first:
			// spread or enclose it, count it, judge it, then charge it. The throttle is outermost so
			// a shadow-banned caller keeps hitting the same 429s everyone else does — a
			// caller that is never throttled again has been told it is banned.

			// Right against the rule, inside the shadow ban: a dropped click never
			// reaches the rule, so it spreads and encloses nothing either. It is counted
			// as one click however many tiles it took.
			var clickUseCase click_usecase.IUseCase = click_usecase.New(tilesChecker, writer, countries)
			clickUseCase = spread_click.New(clickUseCase, spreads, geography, writer, registry)

			clickUseCase = enclose_click.New(clickUseCase, enclosures,
				bonuses.NewTerrain(geography, tilesStorage),
				enclose_click.NewAnnexer(writer, prom_enclose.New(registry, props.Metrics)))

			clickUseCase = prom_click.New(clickUseCase, props.Metrics)

			// The shadow ban. The metric names and the words of the ban line are the
			// observer's; the guard measures and judges.
			guard, err := antibot.New(config.AntiBot, clock, antibot_click.NewObserver(props.Logger, props.Metrics))
			if err != nil {
				return fmt.Errorf("failed to build the antibot guard: %w", err)
			}

			// With the antibot off the guard drops nothing: the click passes, BanPlayer
			// refuses, FindPlayers says nothing of bans and a bomb is never a dud.
			// On, it connects to its own schema here, and its runner closes that pool after the last flush.
			if err := guard.LoadState(ctx); err != nil {
				return fmt.Errorf("failed to load the antibot state: %w", err)
			}
			props.Runners.Add(guard)

			clickUseCase = antibot_click.New(clickUseCase, guard, tilesStorage, clock, props.Metrics)

			// Inside the throttle: presence is what a caller actually managed to do,
			// not what they attempted.
			clickUseCase = bonus_click.New(clickUseCase, registry)

			clickUseCase = throttle_click.New(clickUseCase, limiter, pricer)

			// Outside the throttle: a loop's timing is only whole before it drops clicks.
			clickUseCase = antibot_attempt_click.New(clickUseCase, guard, clock)

			// ---- Admin service ----

			// A quarter of a subscriber's buffer per batch leaves room for the clicks still arriving.
			adminBatch := config.TilesStorage.SubscriberBuffer / 4
			if adminBatch <= 0 {
				adminBatch = 256
			}
			pace := clicks.Pacing{Batch: adminBatch, Pause: 50 * time.Millisecond}

			adminService := planetv1controller.AdminService{
				ReassignCountryHandler: reassign_country_handler.New(audit_reassign.New(
					reassign_country_usecase.New(tilesStorage, countries, pace), props.Logger)),
				FindPlayersHandler: find_players_handler.New(
					find_players_usecase.New(takings, tilesStorage, borders, guard, countries)),
				TopPlayersHandler: top_players_handler.New(top_players_usecase.New(takings, tilesStorage, guard)),
				BanPlayerHandler:  ban_player_handler.New(audit_ban.New(ban_player_usecase.New(guard), props.Logger)),
				RevertPlayerHandler: revert_player_handler.New(
					audit_revert.New(revert_player_usecase.New(takings, tilesStorage, pace), props.Logger)),
				InspectPlayerHandler: inspect_player_handler.New(inspect_player_usecase.New(guard)),
				PaintRandomTilesHandler: paint_random_tiles_handler.New(audit_paint_random.New(
					paint_random_tiles_usecase.New(borders, geography, tilesStorage, countries, clicks.SystemRandom{}, pace),
					props.Logger)),
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

			// This context builds its own verifier from the same `auth:` block the
			// auth context mints with — same secret, same MAC — so neither module
			// has to hand the other an object. Skipped when auth is off.
			if config.Auth.Enabled {
				verifier, err := cpsession.NewSigner(config.Auth)
				if err != nil {
					return fmt.Errorf("failed to build the click session verifier: %w", err)
				}

				interceptors = append(interceptors, planetv1controller.NewSessionInterceptor(
					verifier,
					clock,
					config.Auth.Enforce,
					props.Metrics))
			}

			// ---- Bonus use cases ----

			// The claim also hands the registry its counters: offered against caught
			// is the only way to see whether the pacing and the flight time are set
			// anywhere near right. What each caller did with their box goes to the
			// guard as well, for the catcher watchdog.
			claimBonus, counters := prom_claim_bonus.New(
				claim_bonus_usecase.New(registry, limiter, pricer, spreads, bombs, bombRules.Radius, enclosures, clock),
				props.Metrics)

			registry.Observe(bonuses.Report{
				Offered: counters.Offered.Inc,
				Lapsed: func(scope string) {
					counters.Lapsed.Inc()
					guard.Missed(scope)
				},
				Caught: func(scope string, after time.Duration) {
					counters.Caught.Observe(after.Seconds())
					guard.Caught(scope, after)
				},
			})

			dropped := prom_drop_bomb.New(
				drop_bomb_usecase.New(bombs, registry, geography, tilesStorage, countries, bombRules), props.Metrics)

			// Outside the count, so it can tell the counter a drop was a dud.
			dropBomb := antibot_drop_bomb.New(dropped, guard)

			// ---- Click service ----

			// Each use case is handed only what it reads or writes, which is why the
			// storage appears three times here rather than once as a single object the
			// service holds: the map reader, the subscription and the tile writer are
			// three ports that happen to be served by one adapter.
			service := planetv1controller.ClickService{
				ClickHandler:      click_handler.New(clickUseCase),
				GetBudgetHandler:  get_budget_handler.New(get_budget_usecase.New(limiter, pricer)),
				MapDensityHandler: map_density_handler.New(map_density_usecase.New(tilesChecker)),
				GetMapHandler: get_map_handler.New(
					antibot_get_map.New(get_map_usecase.New(tilesChecker, tilesStorage), guard, tilesChecker)),
				ListenForEventsHandler: listen_for_events_handler.New(antibot_listen_for_events.New(
					listen_for_events_usecase.New(tilesStorage, props.Server.StreamHeartbeat, registry), guard)),
				ClaimBonusHandler: claim_bonus_handler.New(claimBonus),
				DropBombHandler:   drop_bomb_handler.New(dropBomb),
			}

			return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return planetv1connect.NewClickServiceHandler(service, options...)
			}, interceptors...)
		},
	}
}
