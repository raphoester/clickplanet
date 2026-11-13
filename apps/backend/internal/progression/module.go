package progression

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1/progressionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts/rpc_player_accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures/postgres_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures/usecases/decide_departure_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures/usecases/forget_departure_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales/postgres_fight_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales/rpc_scoring_finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales/usecases/forget_fights_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales/usecases/record_finale_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates/postgres_nameplate_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates/usecases/forget_nameplate_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates/usecases/keep_nameplate_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/admin_get_progress_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/admin_get_progress_handler/admin_progress_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_badges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_badges_handler/badges_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_fronts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_progress_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_progress_handler/progress_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_titles_handler/titles_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/reconcile_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/rpc_planet_fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/rpc_planet_fronts/caching_fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/wear_title_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/postgres_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/forget_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/bomb_landed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/bonus_claimed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/fortified_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/message_sent_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/missions_paid_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/player_created_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/player_forgotten_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/player_leaving_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/player_returned_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/quiz_answered_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/season_ended_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/stats_changed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/tiles_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/inprocess_title_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/postgres_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/usecases/award_titles_usecase/notifying_award_titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/usecases/forget_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/usecases/listen_for_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/usecases/reconcile_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/usecases/reconcile_titles_usecase/audit_reconcile_titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/wearing/postgres_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/wearing/usecases/forget_worn_title_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/wearing/usecases/wear_title_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "progression"

const frontsKeptFor = 30 * time.Second

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config, props)
		},
	}
}

func build(ctx context.Context, config Config, props cpbootstrap.Props) error {
	clock := cptime.SystemClock{}

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the progression module asks player about accounts, planet about takes and scoring about finales: %w", err)
	}
	accounts := rpc_player_accounts.New(playerv1connect.NewInternalServiceClient(internal, baseURL))
	fronts := rpc_planet_fronts.New(planetv1connect.NewInternalServiceClient(internal, baseURL))
	scoring := rpc_scoring_finales.New(scoringv1connect.NewInternalServiceClient(internal, baseURL))

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the progression module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the progression module to redis: %w", err)
	}

	statsStore := postgres_stats_store.New(db)
	fightStore := postgres_fight_store.New(db)
	titleStore := postgres_title_store.New(db)
	wornTitleStore := postgres_worn_title_store.New(db)
	nameplateStore := postgres_nameplate_store.New(db)
	departureStore := postgres_departure_store.New(db)

	careers := titles.NewCareers(accounts, fightStore)
	catalog := titles.NewCatalog()
	titleBook := titles.NewBook(titleStore, catalog)
	titleFeed := inprocess_title_feed.New()
	award := notifying_award_titles.New(award_titles_usecase.New(statsStore, careers, titleBook, clock), titleFeed)
	wardrobe := wearing.NewWardrobe(wornTitleStore, titleBook, catalog)
	titleCards := inprocess_title_catalog.New(catalog)
	recordStats := record_stats_usecase.New(statsStore, streams)
	decideDeparture := decide_departure_usecase.New(departureStore)

	var subscribeErr error
	subscribed := func(runner cpbootstrap.Runner, err error) cpbootstrap.Runner {
		subscribeErr = errors.Join(subscribeErr, err)
		return runner
	}
	runners := []cpbootstrap.Runner{
		subscribed(cpredis.SubscribeBatch(ctx, streams, "player-stats", log_subscriber.NewBatch(tiles_taken_subscriber.New(recordStats), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "player-stats-messages", log_subscriber.New(message_sent_subscriber.New(recordStats), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "player-stats-bombs", log_subscriber.New(bomb_landed_subscriber.New(recordStats), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "player-stats-fortifies", log_subscriber.New(fortified_subscriber.New(recordStats), props.Logger))),
		// The offers tell boxes since 2026-11-11, and the quizzes answers since 2026-11-13: each group handles what the
		// stream before held first.
		subscribed(cpredis.SubscribeMoved(ctx, streams, "planet.v1.BonusClaimed", "player-stats-bonuses",
			log_subscriber.New(bonus_claimed_subscriber.New(recordStats), props.Logger))),
		subscribed(cpredis.SubscribeMoved(ctx, streams, "bonuses.v1.QuizAnswered", "player-stats-quizzes",
			log_subscriber.New(quiz_answered_subscriber.New(recordStats), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "player-stats-missions", log_subscriber.New(missions_paid_subscriber.New(recordStats), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "player-titles", log_subscriber.New(stats_changed_subscriber.New(award), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "player-finale-fights", log_subscriber.New(season_ended_subscriber.New(record_finale_usecase.New(scoring, fightStore, award)), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "progression-nameplates", log_subscriber.New(player_created_subscriber.New(keep_nameplate_usecase.New(nameplateStore)), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "progression-departures-leaving", log_subscriber.New(player_leaving_subscriber.New(decideDeparture), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "progression-departures-returns", log_subscriber.New(player_returned_subscriber.New(decideDeparture), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "progression-stats-forgotten", log_subscriber.New(player_forgotten_subscriber.New(forget_stats_usecase.New(statsStore)), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "progression-titles-forgotten", log_subscriber.New(player_forgotten_subscriber.New(forget_titles_usecase.New(titleStore)), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "progression-wearing-forgotten", log_subscriber.New(player_forgotten_subscriber.New(forget_worn_title_usecase.New(wornTitleStore)), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "progression-finales-forgotten", log_subscriber.New(player_forgotten_subscriber.New(forget_fights_usecase.New(fightStore)), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "progression-nameplates-forgotten", log_subscriber.New(player_forgotten_subscriber.New(forget_nameplate_usecase.New(nameplateStore)), props.Logger))),
		subscribed(cpredis.Subscribe(ctx, streams, "progression-departures-forgotten", log_subscriber.New(player_forgotten_subscriber.New(forget_departure_usecase.New(departureStore)), props.Logger))),
	}
	if subscribeErr != nil {
		_ = streams.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the progression module: %w", subscribeErr)
	}
	// Not closers: closers run before the runners stop, and the runners use both.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger, runners...)))

	progressionService := progressionv1controller.ProgressionService{
		GetTitlesHandler: get_titles_handler.New(titles_query.NewPostgresQuery(db, titleCards, clock)),
		WearTitleHandler: wear_title_handler.New(wear_title_usecase.New(wardrobe, clock)),
		GetFrontsHandler: get_fronts_handler.New(fronts),
		GetBadgesHandler: get_badges_handler.New(badges_query.NewPostgresQuery(db, titleCards, clock)),
		GetProgressHandler: get_progress_handler.New(progress_query.NewPostgresQuery(
			db, titleCards, caching_fronts.New(fronts, frontsKeptFor, clock), clock,
		)),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_titles_usecase.New(titleFeed, catalog, props.Server.StreamHeartbeat),
		),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return progressionv1connect.NewProgressionServiceHandler(progressionService, options...)
	}, progressionv1controller.NewSessionInterceptor(props.Sessions, clock, props.Metrics),
		progressionv1controller.NewStreamSessionReader(props.Sessions, clock)); err != nil {
		return fmt.Errorf("failed to mount progression.v1.ProgressionService: %w", err)
	}

	adminService := progressionv1controller.AdminService{
		ReconcileTitlesHandler: reconcile_titles_handler.New(audit_reconcile_titles.New(
			reconcile_titles_usecase.New(statsStore, careers, titleStore, catalog, clock), props.Logger,
		)),
		AdminGetProgressHandler: admin_get_progress_handler.New(
			admin_progress_query.NewPostgresQuery(db, titleCards, fronts, clock),
		),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return progressionv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount progression.v1.AdminService: %w", err)
	}

	props.Logger.Info("progression built", slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Database cppg.Config
	Redis    cpredis.Config
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("progression.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("progression.redis: %w", err)
	}
	return nil
}
