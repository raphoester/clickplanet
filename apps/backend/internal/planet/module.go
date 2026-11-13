package planet

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/allegiance_facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/audit_release"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/log_allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/postgres_allegiance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/postgres_arena_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/close_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/close_arena_usecase/log_close_arena"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/open_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/open_arena_usecase/log_open_arena"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/inmemory_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/postgres_charge_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/postgres_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/apply_grant_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/drop_bomb_usecase/prom_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/drop_bomb_usecase/refusal_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/get_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/grant_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/grant_charges_usecase/audit_grant_charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/pay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/place_shield_usecase/refusal_place_shield"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/use_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_geodesic_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/postgres_tile_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/allegiance_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click/prom_enclose"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/prom_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/reach_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/refusal_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/spread_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/try_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_budget_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/map_density_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/admit_caller_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/apply_grant_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/refuse_caller_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/anonymize_takes_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/find_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/relay_ledger_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/relay_ledger_usecase/log_relay_ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/replay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase/audit_revert"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/top_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements/postgres_placement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements/usecases/return_placements_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements/usecases/return_placements_usecase/log_return_placements"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/close_arena_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/drop_bomb_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/find_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_handler/allegiance_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_of_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_charge_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_charges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_country_sizes_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_fortresses_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_fortresses_handler/fortresses_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_freeze_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_holdings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_holdings_handler/holdings_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_landmass_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_landmass_handler/landmass_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_landmass_handler/landmass_query/inmemory_tile_holders"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler/changes_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_totals_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_totals_handler/totals_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_history_handler/history_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_recent_map_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_recent_map_history_handler/recent_history_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_replay_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_room_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rule_set_handler/rule_set_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rule_set_names_handler/rule_set_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_takes_by_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_takes_by_country_handler/takes_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territories_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territories_handler/territories_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_tile_holder_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_tile_holder_handler/holder_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_tiles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_tiles_handler/tiles_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/grant_charges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/internal_grant_charges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/map_density_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/open_arena_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/place_shield_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/pledge_allegiance_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/release_allegiance_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/reset_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/revert_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/top_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/use_refill_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals/log_refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals/postgres_refusal_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/postgres_reset_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/usecases/reset_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/usecases/reset_map_usecase/log_reset_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/seeds"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/player_forgotten_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	moduleName = "planet"

	ledgerRelayInterval = time.Second
	factsRelayInterval  = 100 * time.Millisecond

	// How long the wallet knows the id of a grant made at once and not for good: a caller that names its grants by what
	// lives minutes (a box's token) calls again within them, or never.
	briefGrantsKept = 10 * time.Minute
)

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			clock := cptime.SystemClock{}
			countries := cpcountries.New()

			gameMap := embedded_geodesic_map.New(config.GameMap.MaxIndex, props.Logger)

			geography, err := gameMap.LoadGeography()
			if err != nil {
				return fmt.Errorf("failed to load the map geography: %w", err)
			}

			borders, err := gameMap.LoadBorders()
			if err != nil {
				return fmt.Errorf("failed to load the map borders: %w", err)
			}

			bridges, err := gameMap.LoadBridges()
			if err != nil {
				return fmt.Errorf("failed to load the map's bridges: %w", err)
			}

			capitals, err := gameMap.LoadCapitals(borders)
			if err != nil {
				return fmt.Errorf("failed to load the map's capitals: %w", err)
			}
			capitals = capitals.Flags(countries.CheckCountry)

			tilesChecker := clicks.NewBoard(config.GameMap.MaxIndex)

			db := cppg.New(config.Database)
			if err := db.ConnectCtx(ctx); err != nil {
				return fmt.Errorf("failed to connect the planet to postgres: %w", err)
			}
			if err := db.Migrate(ctx, migrations.FS); err != nil {
				_ = db.Close()
				return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
			}

			streams := cpredis.New(config.Redis, props.Metrics)
			if err := streams.ConnectCtx(ctx); err != nil {
				_ = db.Close()
				return fmt.Errorf("failed to connect the planet to redis: %w", err)
			}
			closeStores := func() {
				_ = streams.Close()
				_ = db.Close()
			}

			arenaStore := postgres_arena_store.New(db)
			playing, err := arenaStore.Playing(ctx)
			if err != nil {
				closeStores()
				return fmt.Errorf("failed to read the arena open: %w", err)
			}
			inForce := cprules.NewInForce[*planetv1.Rules](clock)
			chargeRules := cprules.Reader(inForce, charges.Part)
			hands := charges.NewCharges(postgres_charge_store.New(db), chargeRules)
			placementStore := postgres_placement_store.New(db)
			escrow := placements.NewEscrow(db, hands, placementStore, uuid_id_provider.Provider{}, clock)
			// Before any request, and a failure only waits for the next boot: a row this boot made is a shield not flushed yet.
			_, _ = log_return_placements.New(return_placements_usecase.New(placementStore, escrow), props.Logger).Execute(ctx)

			capitalPlay := clicks.NewCapitalPlay(capitals, cprules.Reader(inForce, clicks.CapitalsPart))
			tilesStorage := inmemory_tile_storage.New(borders, capitalPlay, config.TilesStorage, postgres_tile_store.New(db), db,
				escrow, clock, props.Logger)
			if err := tilesStorage.Load(ctx, playing.Played()); err != nil {
				closeStores()
				return fmt.Errorf("failed to load the tile map: %w", err)
			}

			ledgerStore := postgres_ledger_store.New(db)
			takings := inmemory_ledger_storage.New(config.LedgerStorage, ledgerStore, props.Logger)
			if err := takings.Load(ctx); err != nil {
				closeStores()
				return fmt.Errorf("failed to load the ledger: %w", err)
			}

			ruleParts := parts.All()
			ruleSets := cprules.NewPostgresStore(db, ruleParts)
			play := cprules.NewPlayUseCase(ruleSets, cprules.NewPostgresChoices(db), inForce, nil)
			playRules := cprules.NewLoggedPlay(play, props.Logger,
				cprules.PlayLog{Taken: "rules in force", Failed: "failed to play the rules"})
			if _, _, err := playRules.Execute(ctx); err != nil {
				closeStores()
				return fmt.Errorf("failed to play the rules postgres keeps: %w", err)
			}
			resetStore := postgres_reset_store.New(db)
			throttle := cprules.Reader(inForce, clicks.ThrottlePart)
			freeze := cprules.Reader(inForce, clicks.FreezePart)
			lock := cprules.Reader(inForce, allegiances.LockPart)

			inPlay := arenas.NewInPlay(clock)
			inPlay.Resume(playing)
			reach := clicks.NewReach(cprules.Reader(inForce, clicks.ReachPart), geography, bridges, tilesStorage)
			setups := arenas.NewSetups(ruleSets, seeds.NewCatalog(seeds.Ground{Capitals: capitals}))

			grantStore := postgres_grant_store.New(db)
			board := notices.New()

			grants, err := cpredis.Handle(ctx, streams, log_subscriber.New(apply_grant_handler.New(
				apply_grant_usecase.New(db, grantStore, hands, board, clock)), props.Logger))
			if err != nil {
				closeStores()
				return fmt.Errorf("failed to handle planet.v1.GrantCharges: %w", err)
			}
			deletions, err := cpredis.Subscribe(ctx, streams, "planet-ledger-forgotten",
				log_subscriber.New(player_forgotten_subscriber.New(
					anonymize_takes_usecase.New(takings, ledgerStore, tilesStorage)), props.Logger))
			if err != nil {
				closeStores()
				return fmt.Errorf("failed to subscribe the ledger to player.v1.PlayerForgotten: %w", err)
			}

			relay := relay_ledger_usecase.NewRunner(ledgerRelayInterval, log_relay_ledger.New(
				relay_ledger_usecase.New(takings, ledgerStore, borders, geography, streams), props.Logger))

			refusalBook := refusals.NewBook(postgres_refusal_store.New(db), clock)
			refused, err := cpredis.Handle(ctx, streams, log_subscriber.New(refuse_caller_handler.New(refusalBook), props.Logger))
			if err != nil {
				closeStores()
				return fmt.Errorf("failed to handle planet.v1.RefuseCaller: %w", err)
			}
			admitted, err := cpredis.Handle(ctx, streams, log_subscriber.New(admit_caller_handler.New(refusalBook), props.Logger))
			if err != nil {
				closeStores()
				return fmt.Errorf("failed to handle planet.v1.AdmitCaller: %w", err)
			}
			refusing := log_refusals.New(refusalBook, props.Logger)

			told := facts.NewLog(clock)

			// Not closers: closers run before the runners' last flush, and the relays' last send.
			props.Runners.Add(cppg.CloseAfter(db, props.Logger, tilesStorage, takings,
				cpredis.CloseAfter(streams, props.Logger, deletions, relay, grants, refused, admitted,
					facts.NewRelay(told, streams, factsRelayInterval))))
			props.Runners.Add(ledger.NewRetention(config.Ledger, takings, clock))

			limiter := cpratelimit.NewRuled("click-limiter",
				func() cpratelimit.Policy { return throttle().Policy(inForce.Since()) }, config.RateLimiter.SweepInterval, clock)
			props.Runners.Add(limiter)
			buckets := clicks.NewBuckets(throttle, clock)

			pricer := clicks.NewToll(cprules.Reader(inForce, clicks.TollPart), tilesStorage)

			book := allegiances.NewBook(postgres_allegiance_store.New(db), countries, lock, tilesStorage, clock, allegiance_facts.New(told))
			flags := log_allegiances.New(book, props.Logger)

			writer := ledger.NewRecording(tilesStorage,
				clicks.NewClaiming(tilesStorage, func() int { return chargeRules().Shield.PerTile }), takings, clock)

			bombing := charges.NewBombing(chargeRules, geography.Spacing())

			var clickUseCase click_usecase.IUseCase = click_usecase.New(tilesChecker, writer, countries)
			clickUseCase = spread_click.New(clickUseCase, hands, reach, writer, board)

			clickUseCase = enclose_click.New(clickUseCase, hands,
				charges.NewTerrain(geography, tilesStorage),
				enclose_click.NewAnnexer(prom_enclose.New(writer, props.Metrics), hands),
				chargeRules)

			clickUseCase = prom_click.New(clickUseCase, props.Metrics)

			clickUseCase = throttle_click.New(clickUseCase, limiter, pricer, buckets)

			// Outside the throttle, so a click out of the flag's reach in an arena spends nothing.
			clickUseCase = reach_click.New(clickUseCase, reach)

			// Outside the throttle, so a click refused for its flag spends nothing.
			clickUseCase = allegiance_click.New(clickUseCase, flags)

			// Outside the throttle: a refused click must not spend a token, or the client shows a 429.
			clickUseCase = refusal_click.New(clickUseCase, refusing, props.Metrics)

			clickUseCase = try_click.New(clickUseCase, tilesStorage, geography, throttle, told)

			adminBatch := config.TilesStorage.SubscriberBuffer / 4
			if adminBatch <= 0 {
				adminBatch = 256
			}
			pace := clicks.Pacing{Batch: adminBatch, Pause: 50 * time.Millisecond}

			getTerritories := get_territories_handler.New(
				territories_query.NewMemoryQuery(tilesStorage, borders, config.GameMap.MaxIndex))
			getLedger := get_ledger_handler.New(changes_query.NewPostgresQuery(db, clock, arenaStore))
			adminService := planetv1controller.AdminService{
				FindPlayersHandler: find_players_handler.New(
					find_players_usecase.New(takings, tilesStorage, borders, refusalBook, countries)),
				TopPlayersHandler: top_players_handler.New(top_players_usecase.New(takings, tilesStorage, refusalBook)),
				RevertPlayerHandler: revert_player_handler.New(
					audit_revert.New(revert_player_usecase.New(takings, tilesStorage, arenaStore, pace), props.Logger)),
				GrantChargesHandler: grant_charges_handler.New(
					audit_grant_charges.New(grant_charges_usecase.New(hands, board), props.Logger)),
				ReleaseAllegianceHandler: release_allegiance_handler.New(audit_release.New(book, props.Logger), lock),
				GetReplayHandler:         get_replay_handler.New(replay_usecase.New(takings, resetStore, arenaStore, tilesStorage, clock)),
				GetLedgerHandler:         getLedger,
				GetTerritoriesHandler:    getTerritories,
				GetRulesHandler:          get_rules_handler.New(rule_sets_query.NewPostgresQuery(db), inForce),
				SaveRuleSetHandler: save_rule_set_handler.New(
					cprules.NewAuditedSave(cprules.NewSaveUseCase(db, ruleSets, ruleParts, playRules), props.Logger,
						cprules.SaveLog{Saved: "admin rule set saved", Refused: "admin rule set refused"})),
			}

			if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return planetv1connect.NewAdminServiceHandler(adminService, options...)
			}); err != nil {
				return err
			}

			freezing := func() clicks.Freeze { return freeze() || inPlay.Freeze() }
			internalService := planetv1controller.InternalService{
				PlayRuleSetHandler: play_rule_set_handler.New(playRules),
				ResetMapHandler: reset_map_handler.New(
					log_reset_map.New(reset_map_usecase.New(tilesStorage, resetStore), props.Logger)),
				OpenArenaHandler: open_arena_handler.New(
					log_open_arena.New(open_arena_usecase.New(tilesStorage, setups, arenaStore, inPlay, clock), props.Logger)),
				CloseArenaHandler: close_arena_handler.New(
					log_close_arena.New(close_arena_usecase.New(tilesStorage, arenaStore, inPlay, clock), props.Logger)),
				GetLedgerHandler:         getLedger,
				GetLedgerTotalsHandler:   get_ledger_totals_handler.New(totals_query.NewPostgresQuery(db)),
				GetMapHistoryHandler:     get_map_history_handler.New(history_query.NewPostgresQuery(db, arenaStore)),
				GetTakesByCountryHandler: get_takes_by_country_handler.New(takes_query.NewPostgresQuery(db)),
				GetTerritoriesHandler:    getTerritories,
				GetCountrySizesHandler:   get_country_sizes_handler.New(borders),
				GetAllegianceOfHandler:   get_allegiance_of_handler.New(allegiance_query.NewPostgresQuery(db, lock)),
				GetRuleSetNamesHandler:   get_rule_set_names_handler.New(rule_set_names_query.NewPostgresQuery(db)),
				GetRuleSetHandler:        get_rule_set_handler.New(rule_set_query.NewPostgresQuery(db)),
				// A grant made at once is known by its id in memory for a while, or in charge_grants for good.
				GrantChargesHandler: internal_grant_charges_handler.New(
					pay_usecase.New(db, inmemory_grant_store.New(briefGrantsKept), hands, board, clock),
					pay_usecase.New(db, grantStore, hands, board, clock)),
				GetRoomHandler:   get_room_handler.New(hands, chargeRules),
				GetFreezeHandler: get_freeze_handler.New(freezing),
				GetTilesHandler:  get_tiles_handler.New(tilesChecker, tiles_query.NewMemoryQuery(tilesStorage, geography)),
			}
			if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return planetv1connect.NewInternalServiceHandler(internalService, options...)
			}); err != nil {
				return fmt.Errorf("failed to mount planet.v1.InternalService: %w", err)
			}

			blocklist := cpipblock.New(config.VPNBlocklist)
			if err := blocklist.Load(); err != nil {
				return fmt.Errorf("failed to load the vpn blocklist: %w", err)
			}

			if sizes := blocklist.Sizes(); len(sizes) > 0 {
				props.Logger.Info("vpn blocklist enabled", slog.Any("ranges", sizes))
			}

			interceptors := []connect.Interceptor{
				planetv1controller.NewCacheInterceptor(tilesStorage),
				planetv1controller.NewVPNBlockInterceptor(blocklist, props.Metrics),
				planetv1controller.NewFreezeInterceptor(freezing),
			}

			if config.Auth.Enabled {
				interceptors = append(interceptors,
					planetv1controller.NewSessionInterceptor(props.Sessions, clock, props.Metrics),
					planetv1controller.NewSessionReaderInterceptor(props.Sessions, clock))
			}

			dropBomb := prom_drop_bomb.New(
				refusal_drop_bomb.New(drop_bomb_usecase.New(hands, geography, writer, countries, bombing), refusing),
				props.Metrics)

			service := planetv1controller.ClickService{
				ClickHandler:       click_handler.New(clickUseCase),
				GetBudgetHandler:   get_budget_handler.New(get_budget_usecase.New(limiter, pricer, buckets, freeze)),
				MapDensityHandler:  map_density_handler.New(map_density_usecase.New(tilesChecker)),
				GetMapHandler:      get_map_handler.New(get_map_usecase.New(tilesChecker, tilesStorage, told)),
				GetHoldingsHandler: get_holdings_handler.New(holdings_query.NewMemoryQuery(tilesStorage)),
				ListenForEventsHandler: listen_for_events_handler.New(
					listen_for_events_usecase.New(tilesStorage, props.Server.StreamHeartbeat, board, inForce, told)),
				DropBombHandler:   drop_bomb_handler.New(dropBomb),
				UseRefillHandler:  use_refill_handler.New(use_refill_usecase.New(hands, limiter, pricer, buckets)),
				GetChargesHandler: get_charges_handler.New(get_charges_usecase.New(hands)),
				GetChargeRulesHandler: get_charge_rules_handler.New(chargeRules, bombing, pricer,
					cprules.Reader(inForce, clicks.ReachPart), cprules.Reader(inForce, clicks.CapitalsPart)),
				PlaceShieldHandler: place_shield_handler.New(refusal_place_shield.New(
					place_shield_usecase.New(placements.NewGate(tilesStorage, countries, flags, chargeRules), escrow, writer),
					refusing)),
				GetFortressesHandler: get_fortresses_handler.New(fortresses_query.NewMemoryQuery(tilesStorage)),
				GetTileHolderHandler: get_tile_holder_handler.New(tilesChecker, holder_query.NewPostgresQuery(db, arenaStore, clock)),
				GetLandmassHandler: get_landmass_handler.New(borders,
					landmass_query.NewPostgresQuery(db, inmemory_tile_holders.New(tilesStorage), arenaStore, clock)),
				GetRecentMapHistoryHandler: get_recent_map_history_handler.New(
					recent_history_query.NewPostgresQuery(db, tilesStorage, arenaStore, resetStore, clock), clock),
				GetAllegianceHandler:    get_allegiance_handler.New(allegiance_query.NewPostgresQuery(db, lock)),
				PledgeAllegianceHandler: pledge_allegiance_handler.New(book),
			}

			return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return planetv1connect.NewClickServiceHandler(service, options...)
			}, interceptors...)
		},
	}
}
