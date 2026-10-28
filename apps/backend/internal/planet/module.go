package planet

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/postgres_allegiance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/box_facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/postgres_charge_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/postgres_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase/prom_answer_quiz"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/apply_grant_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase/prom_claim_bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase/prom_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase/refusal_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/get_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_charges_usecase/audit_grant_charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/open_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase/refusal_place_shield"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/use_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_geodesic_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/postgres_tile_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/allegiance_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/bonus_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click/prom_enclose"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/prom_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/refusal_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/spread_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/try_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_budget_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/map_density_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase/audit_paint_random"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase/audit_reassign"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/admit_caller_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/apply_grant_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/refuse_caller_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/reset_map_handler"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/answer_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/drop_bomb_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/find_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_handler/allegiance_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_of_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_bonus_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_charges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_country_sizes_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_fortresses_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_fortresses_handler/fortresses_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_holdings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_holdings_handler/holdings_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_landmass_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_landmass_handler/landmass_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_landmass_handler/landmass_query/inmemory_tile_holders"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler/changes_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_replay_handler"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/grant_charges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/map_density_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/open_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/paint_random_tiles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/place_shield_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/pledge_allegiance_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/reassign_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/revert_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/top_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/use_refill_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals/log_refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals/postgres_refusal_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/postgres_reset_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/usecases/reset_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/usecases/reset_map_usecase/log_reset_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_choice_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/play_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/play_rules_usecase/log_play_rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/save_rule_set_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/save_rule_set_usecase/audit_save_rule_set"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	moduleName = "planet"

	ledgerRelayInterval = time.Second
	factsRelayInterval  = 100 * time.Millisecond
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

			tilesStorage := inmemory_tile_storage.New(borders, config.TilesStorage, postgres_tile_store.New(db), clock, props.Logger)
			if err := tilesStorage.Load(ctx); err != nil {
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
			ruleSets := postgres_rule_set_store.New(db, ruleParts)
			inForce := rules.NewInForce(clock)
			play := play_rules_usecase.New(ruleSets, postgres_choice_store.New(db), inForce)
			playRules := log_play_rules.New(play, props.Logger)
			if _, _, err := playRules.Execute(ctx); err != nil {
				closeStores()
				return fmt.Errorf("failed to play the rules postgres keeps: %w", err)
			}
			chosen, err := cpredis.Handle(ctx, streams, play_rule_set_handler.New(playRules))
			if err != nil {
				closeStores()
				return fmt.Errorf("failed to handle planet.v1.PlayRuleSet: %w", err)
			}
			resetStore := postgres_reset_store.New(db)
			resetMap := log_reset_map.New(reset_map_usecase.New(tilesStorage, resetStore), props.Logger)
			resetting, err := cpredis.Handle(ctx, streams, reset_map_handler.New(resetMap))
			if err != nil {
				closeStores()
				return fmt.Errorf("failed to handle planet.v1.ResetMap: %w", err)
			}
			throttle := rules.Reader(inForce, clicks.ThrottlePart)
			game := rules.Reader(inForce, bonuses.Part)
			freeze := rules.Reader(inForce, clicks.FreezePart)

			charges := bonuses.NewCharges(postgres_charge_store.New(db), game)

			grants, err := cpredis.Handle(ctx, streams, log_subscriber.New(apply_grant_handler.New(
				apply_grant_usecase.New(db, postgres_grant_store.New(db), charges, clock)), props.Logger))
			if err != nil {
				closeStores()
				return fmt.Errorf("failed to handle planet.v1.GrantCharges: %w", err)
			}

			deletions, err := cpredis.Subscribe(ctx, streams, "planet-ledger-accounts",
				log_subscriber.New(account_deleted_subscriber.New(
					anonymize_takes_usecase.New(takings, ledgerStore, tilesStorage)), props.Logger))
			if err != nil {
				closeStores()
				return fmt.Errorf("failed to subscribe the ledger to auth.v1.AccountDeleted: %w", err)
			}

			relay := relay_ledger_usecase.NewRunner(ledgerRelayInterval, log_relay_ledger.New(
				relay_ledger_usecase.New(takings, ledgerStore, borders, streams), props.Logger))

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
				cpredis.CloseAfter(streams, props.Logger, deletions, relay, grants, refused, admitted, chosen, resetting,
					facts.NewRelay(told, streams, factsRelayInterval))))
			props.Runners.Add(ledger.NewRetention(config.Ledger, takings, clock))

			limiter := cpratelimit.NewRuled("click-limiter",
				func() cpratelimit.Policy { return throttle().Policy(inForce.Since()) }, config.RateLimiter.SweepInterval, clock)
			props.Runners.Add(limiter)
			buckets := clicks.NewBuckets(throttle)

			pricer := clicks.NewToll(rules.Reader(inForce, clicks.TollPart), tilesStorage)

			book := allegiances.NewBook(postgres_allegiance_store.New(db), countries, clock)

			writer := ledger.NewRecording(tilesStorage,
				clicks.NewClaiming(tilesStorage, func() int { return game().Shield.PerTile }), takings, clock)

			bank, err := quizzes.Load(tilesStorage)
			if err != nil {
				return fmt.Errorf("failed to load the quiz bank: %w", err)
			}
			props.Logger.Info("quiz bank loaded",
				slog.String("bank", bank.Version()),
				slog.Int("questions", bank.Size()),
				slog.Int("subjects", bank.Subjects()))

			registry := bonuses.New(config.Bonus, clock, charges, game, bank)
			props.Runners.Add(registry)

			bombing := bonuses.NewBombing(game, geography.Spacing())

			var clickUseCase click_usecase.IUseCase = click_usecase.New(tilesChecker, writer, countries)
			clickUseCase = spread_click.New(clickUseCase, charges, geography, writer, registry)

			clickUseCase = enclose_click.New(clickUseCase, charges,
				bonuses.NewTerrain(geography, tilesStorage),
				enclose_click.NewAnnexer(writer, charges, prom_enclose.New(registry, props.Metrics)),
				game)

			clickUseCase = prom_click.New(clickUseCase, props.Metrics)

			clickUseCase = throttle_click.New(clickUseCase, limiter, pricer, buckets)

			// Outside the throttle, so it reads the bank the click left.
			clickUseCase = bonus_click.New(clickUseCase, registry)

			// Outside the throttle, so a click refused for its flag spends nothing.
			clickUseCase = allegiance_click.New(clickUseCase, book)

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
			getLedger := get_ledger_handler.New(changes_query.NewPostgresQuery(db, clock))
			adminService := planetv1controller.AdminService{
				ReassignCountryHandler: reassign_country_handler.New(audit_reassign.New(
					reassign_country_usecase.New(tilesStorage, countries, pace), props.Logger)),
				FindPlayersHandler: find_players_handler.New(
					find_players_usecase.New(takings, tilesStorage, borders, refusalBook, countries)),
				TopPlayersHandler: top_players_handler.New(top_players_usecase.New(takings, tilesStorage, refusalBook)),
				RevertPlayerHandler: revert_player_handler.New(
					audit_revert.New(revert_player_usecase.New(takings, tilesStorage, pace), props.Logger)),
				PaintRandomTilesHandler: paint_random_tiles_handler.New(audit_paint_random.New(
					paint_random_tiles_usecase.New(borders, geography, tilesStorage, countries, clicks.SystemRandom{}, pace),
					props.Logger)),
				GrantChargesHandler: grant_charges_handler.New(
					audit_grant_charges.New(grant_charges_usecase.New(charges), props.Logger)),
				GetReplayHandler:      get_replay_handler.New(replay_usecase.New(takings, resetStore, tilesChecker, tilesStorage, clock)),
				GetLedgerHandler:      getLedger,
				GetTerritoriesHandler: getTerritories,
				GetRulesHandler:       get_rules_handler.New(rule_sets_query.NewPostgresQuery(db), inForce),
				SaveRuleSetHandler: save_rule_set_handler.New(
					audit_save_rule_set.New(save_rule_set_usecase.New(ruleSets, ruleParts, playRules), props.Logger)),
			}

			if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return planetv1connect.NewAdminServiceHandler(adminService, options...)
			}); err != nil {
				return err
			}

			internalService := planetv1controller.InternalService{
				GetLedgerHandler:         getLedger,
				GetTakesByCountryHandler: get_takes_by_country_handler.New(takes_query.NewPostgresQuery(db)),
				GetTerritoriesHandler:    getTerritories,
				GetCountrySizesHandler:   get_country_sizes_handler.New(borders),
				GetAllegianceOfHandler:   get_allegiance_of_handler.New(allegiance_query.NewPostgresQuery(db)),
				GetRuleSetNamesHandler:   get_rule_set_names_handler.New(rule_set_names_query.NewPostgresQuery(db)),
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
				planetv1controller.NewFreezeInterceptor(freeze),
			}

			if config.Auth.Enabled {
				verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", "planet")))
				interceptors = append(interceptors,
					planetv1controller.NewSessionInterceptor(verifier, clock, config.Auth.Enforce, props.Metrics),
					planetv1controller.NewSessionReaderInterceptor(verifier, clock))
			}

			claimBonus, counters := prom_claim_bonus.New(
				claim_bonus_usecase.New(registry, charges, streams, clock),
				props.Metrics)

			openQuiz := open_quiz_usecase.New(registry)
			answerQuiz, quizCounters := prom_answer_quiz.New(
				answer_quiz_usecase.New(registry, charges, streams, clock),
				props.Metrics)

			boxes := box_facts.New(told)
			registry.Observe(bonuses.Report{
				Offered: counters.Offered.Inc,
				Lapsed: func(scope string) {
					counters.Lapsed.Inc()
					boxes.Lapsed(scope)
				},
				Caught: func(scope string, after time.Duration) {
					counters.Caught.Observe(after.Seconds())
					boxes.Caught(scope, after)
				},
				Foreign: func(scope string) {
					counters.Foreign.Inc()
					boxes.Foreign(scope)
				},

				QuizOffered: quizCounters.Offered.Inc,
				QuizLapsed:  func(string) { quizCounters.Lapsed.Inc() },
				QuizAnswered: func(_ string, correct bool, after time.Duration) {
					quizCounters.Answered.WithLabelValues(strconv.FormatBool(correct)).Observe(after.Seconds())
				},
			})

			dropBomb := prom_drop_bomb.New(
				refusal_drop_bomb.New(drop_bomb_usecase.New(charges, geography, writer, countries, bombing), refusing),
				props.Metrics)

			service := planetv1controller.ClickService{
				ClickHandler:       click_handler.New(clickUseCase),
				GetBudgetHandler:   get_budget_handler.New(get_budget_usecase.New(limiter, pricer, buckets, freeze)),
				MapDensityHandler:  map_density_handler.New(map_density_usecase.New(tilesChecker)),
				GetMapHandler:      get_map_handler.New(get_map_usecase.New(tilesChecker, tilesStorage, told)),
				GetHoldingsHandler: get_holdings_handler.New(holdings_query.NewMemoryQuery(tilesStorage)),
				ListenForEventsHandler: listen_for_events_handler.New(
					listen_for_events_usecase.New(tilesStorage, props.Server.StreamHeartbeat, registry, inForce, told)),
				ClaimBonusHandler:    claim_bonus_handler.New(claimBonus),
				DropBombHandler:      drop_bomb_handler.New(dropBomb),
				UseRefillHandler:     use_refill_handler.New(use_refill_usecase.New(charges, limiter, pricer, buckets)),
				GetChargesHandler:    get_charges_handler.New(get_charges_usecase.New(charges)),
				GetBonusRulesHandler: get_bonus_rules_handler.New(game, bombing, pricer),
				OpenQuizHandler:      open_quiz_handler.New(openQuiz),
				AnswerQuizHandler:    answer_quiz_handler.New(answerQuiz),
				PlaceShieldHandler: place_shield_handler.New(refusal_place_shield.New(
					place_shield_usecase.New(charges, writer, countries, book, game), refusing)),
				GetFortressesHandler: get_fortresses_handler.New(fortresses_query.NewMemoryQuery(tilesStorage)),
				GetTileHolderHandler: get_tile_holder_handler.New(tilesChecker, holder_query.NewPostgresQuery(db)),
				GetLandmassHandler: get_landmass_handler.New(borders,
					landmass_query.NewPostgresQuery(db, inmemory_tile_holders.New(tilesStorage))),
				GetAllegianceHandler:    get_allegiance_handler.New(allegiance_query.NewPostgresQuery(db)),
				PledgeAllegianceHandler: pledge_allegiance_handler.New(book),
			}

			return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return planetv1connect.NewClickServiceHandler(service, options...)
			}, interceptors...)
		},
	}
}
