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

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/inmemory_event_buffer"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/inmemory_event_buffer/log_flush"
	activitymigrations "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/postgres_event_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/usecases/prune_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/usecases/prune_usecase/log_prune"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/activity_boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/inmemory_charge_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/postgres_charge_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase/prom_answer_quiz"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase/prom_claim_bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase/antibot_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase/prom_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase/publishing_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/get_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/open_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/use_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_geodesic_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/postgres_tile_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/activity_attempt_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/activity_take_click"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase/activity_get_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase/antibot_get_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase/activity_listen_for_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase/antibot_listen_for_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/map_density_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase/audit_paint_random"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase/audit_reassign"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/publishing_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/ban_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/ban_player_usecase/audit_ban"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/find_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/inspect_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase/audit_revert"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/top_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/answer_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/ban_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/drop_bomb_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/find_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_bonus_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_charges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/inspect_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/map_density_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/open_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/paint_random_tiles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/reassign_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/revert_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/rpc_session_verifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/top_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/use_refill_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "planet"

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

			charges := inmemory_charge_storage.New(config.ChargeStorage, config.Bonus.ChargesConfig(),
				postgres_charge_store.New(db), props.Logger)
			if err := charges.Load(ctx); err != nil {
				_ = db.Close()
				return fmt.Errorf("failed to load the charges: %w", err)
			}

			// Not a closer: closers run before the runners' last flush.
			props.Runners.Add(cppg.CloseAfter(db, props.Logger, tilesStorage, takings, charges))
			props.Runners.Add(ledger.NewRetention(config.Ledger, takings, clock))

			var recorder activity.Recorder = activity.Discard{}
			if config.Activity.Enabled {
				activityConfig := config.Activity.WithDefaults()

				events := cppg.New(activityConfig.Database)
				if err := events.ConnectCtx(ctx); err != nil {
					return fmt.Errorf("failed to connect the activity to postgres: %w", err)
				}
				if err := events.Migrate(ctx, activitymigrations.FS); err != nil {
					_ = events.Close()
					return fmt.Errorf("failed to migrate the %s schema: %w", activityConfig.Database.Schema, err)
				}

				store := postgres_event_store.New(events)
				buffer := inmemory_event_buffer.New(activityConfig.MaxPending, store)
				props.Runners.Add(cppg.CloseAfter(events, props.Logger,
					inmemory_event_buffer.NewRunner(activityConfig.FlushInterval, log_flush.New(buffer, props.Logger)),
					prune_usecase.NewRunner(activityConfig.SweepInterval, log_prune.New(
						prune_usecase.New(activityConfig.Retention, activityConfig.MaxEvents, clock, store), props.Logger))))
				recorder = buffer
			}

			limiter := cpratelimit.New("click-limiter", config.RateLimiter.Config, clock)
			props.Runners.Add(limiter)
			buckets := config.RateLimiter.Buckets()

			pricer := clicks.NewToll(config.Toll, tilesStorage)

			homeSoil := clicks.NewHomeSoil(config.HomeSoil, borders)
			if homeSoil.Enabled() {
				props.Logger.Info("home soil enabled: native land takes two clicks")
			}

			writer := ledger.NewRecording(tilesStorage, publishing_ledger_storage.New(takings, props.Events), clock)

			registry := bonuses.New(config.Bonus, clock, charges)
			props.Runners.Add(registry)

			if config.Bonus.Quiz.Enabled {
				bank, err := quizzes.Load(config.Bonus.Quiz, tilesStorage)
				if err != nil {
					return fmt.Errorf("failed to load the quiz bank: %w", err)
				}

				registry.Quizzing(config.Bonus.Quiz, bank)
				props.Logger.Info("quizzes enabled",
					slog.String("bank", bank.Name()),
					slog.Int("questions", bank.Size()),
					slog.Int("subjects", bank.Subjects()))
			}

			bombRules := bonuses.NewBombRules(config.Bonus.Bomb, geography.Spacing())

			var clickUseCase click_usecase.IUseCase = click_usecase.New(tilesChecker, writer, countries, homeSoil)
			clickUseCase = spread_click.New(clickUseCase, charges, geography, writer, homeSoil, registry)

			clickUseCase = enclose_click.New(clickUseCase, charges,
				bonuses.NewTerrain(geography, tilesStorage),
				enclose_click.NewAnnexer(writer, homeSoil, charges, prom_enclose.New(registry, props.Metrics)))

			clickUseCase = prom_click.New(clickUseCase, props.Metrics)

			guard, err := antibot.New(config.AntiBot, clock, antibot_click.NewObserver(props.Logger, props.Metrics))
			if err != nil {
				return fmt.Errorf("failed to build the antibot guard: %w", err)
			}

			if err := guard.LoadState(ctx); err != nil {
				return fmt.Errorf("failed to load the antibot state: %w", err)
			}
			props.Runners.Add(guard)

			// Inside the shadow ban: a click it dropped took no tile.
			clickUseCase = activity_take_click.New(clickUseCase, recorder, tilesStorage, clock)

			clickUseCase = antibot_click.New(clickUseCase, guard, tilesStorage, homeSoil, clock, props.Metrics)

			clickUseCase = bonus_click.New(clickUseCase, registry)

			// Outside the shadow ban, or a banned caller would stop seeing 429s and know.
			clickUseCase = throttle_click.New(clickUseCase, limiter, pricer, buckets)

			clickUseCase = antibot_attempt_click.New(clickUseCase, guard, clock)
			clickUseCase = activity_attempt_click.New(clickUseCase, recorder, tilesStorage, clock)

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
				InspectPlayerHandler: inspect_player_handler.New(inspect_player_usecase.New(guard, takings)),
				PaintRandomTilesHandler: paint_random_tiles_handler.New(audit_paint_random.New(
					paint_random_tiles_usecase.New(borders, geography, tilesStorage, countries, clicks.SystemRandom{}, pace),
					props.Logger)),
			}

			if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return planetv1connect.NewAdminServiceHandler(adminService, options...)
			}); err != nil {
				return err
			}

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

			if config.Auth.Enabled {
				verifier := rpc_session_verifier.New(props.Internal, props.Logger)
				interceptors = append(interceptors,
					planetv1controller.NewSessionInterceptor(verifier, clock, config.Auth.Enforce, props.Metrics),
					planetv1controller.NewSessionReaderInterceptor(verifier, clock))
			}

			claimBonus, counters := prom_claim_bonus.New(
				claim_bonus_usecase.New(registry, charges),
				props.Metrics)

			openQuiz := open_quiz_usecase.New(registry)
			answerQuiz, quizCounters := prom_answer_quiz.New(
				answer_quiz_usecase.New(registry, charges),
				props.Metrics)

			boxes := activity_boxes.New(recorder, clock)
			registry.Observe(bonuses.Report{
				Offered: func(scope string) {
					counters.Offered.Inc()
					boxes.Offered(scope)
				},
				Lapsed: func(scope string) {
					counters.Lapsed.Inc()
					guard.Missed(scope)
					boxes.Lapsed(scope)
				},
				Caught: func(scope string, after time.Duration) {
					counters.Caught.Observe(after.Seconds())
					guard.Caught(scope, after)
					boxes.Caught(scope, after)
				},
				Foreign: func(scope string) {
					counters.Foreign.Inc()
					guard.Foreign(scope)
					boxes.Foreign(scope)
				},

				QuizOffered: quizCounters.Offered.Inc,
				QuizLapsed:  func(string) { quizCounters.Lapsed.Inc() },
				QuizAnswered: func(_ string, correct bool, after time.Duration) {
					quizCounters.Answered.WithLabelValues(strconv.FormatBool(correct)).Observe(after.Seconds())
				},
			})

			dropped := prom_drop_bomb.New(
				publishing_drop_bomb.New(
					drop_bomb_usecase.New(charges, geography, tilesStorage, countries, bombRules),
					borders, props.Events, clock),
				props.Metrics)

			dropBomb := antibot_drop_bomb.New(dropped, guard)

			rules := bonuses.Rules{
				BlastRadius:       bombRules.Radius,
				EnclosureMaxTiles: charges.EnclosureMaxTiles(),
				SpreadClicks:      charges.SpreadClicks(),
				Enclosures:        charges.Enclosures(),
			}

			service := planetv1controller.ClickService{
				ClickHandler:      click_handler.New(clickUseCase),
				GetBudgetHandler:  get_budget_handler.New(get_budget_usecase.New(limiter, pricer, buckets)),
				MapDensityHandler: map_density_handler.New(map_density_usecase.New(tilesChecker)),
				GetMapHandler: get_map_handler.New(activity_get_map.New(
					antibot_get_map.New(get_map_usecase.New(tilesChecker, tilesStorage), guard, tilesChecker),
					recorder, tilesChecker, clock)),
				ListenForEventsHandler: listen_for_events_handler.New(activity_listen_for_events.New(
					antibot_listen_for_events.New(
						listen_for_events_usecase.New(tilesStorage, props.Server.StreamHeartbeat, registry), guard),
					recorder, clock)),
				ClaimBonusHandler:    claim_bonus_handler.New(claimBonus),
				DropBombHandler:      drop_bomb_handler.New(dropBomb),
				UseRefillHandler:     use_refill_handler.New(use_refill_usecase.New(charges, limiter, pricer, buckets)),
				GetChargesHandler:    get_charges_handler.New(get_charges_usecase.New(charges)),
				GetBonusRulesHandler: get_bonus_rules_handler.New(rules, homeSoil),
				OpenQuizHandler:      open_quiz_handler.New(openQuiz),
				AnswerQuizHandler:    answer_quiz_handler.New(answerQuiz),
			}

			return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return planetv1connect.NewClickServiceHandler(service, options...)
			}, interceptors...)
		},
	}
}
