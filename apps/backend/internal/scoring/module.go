package scoring

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar/usecases/get_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/rpc_planet_territories"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_ended_rounds_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_ended_rounds_usecase/log_close_ended_rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/count_take_usecase"
	round_forget_account_usecase "github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/open_own_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/postgres_choice_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/usecases/play_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/usecases/play_rules_usecase/log_play_rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/usecases/save_rule_set_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/usecases/save_rule_set_usecase/audit_save_rule_set"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_hall_of_fame_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_hall_of_fame_handler/hall_of_fame_query"
	hall_of_fame_authors "github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_hall_of_fame_handler/hall_of_fame_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_season_handler/my_season_query"
	my_season_authors "github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_season_handler/my_season_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_race_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_rule_set_names_handler/rule_set_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_standings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_standings_handler/standings_query"
	standings_authors "github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_standings_handler/standings_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/inprocess_board_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/inprocess_board_feed/log_board_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/inprocess_race_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/inprocess_race_feed/log_race_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/race_query"
	race_authors "github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/race_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/round_take_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "scoring"

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
	seasons := calendar.New(config.Calendar)
	countries := cpcountries.New()

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the scoring module asks the player module who plays: %w", err)
	}
	player := playerv1connect.NewInternalServiceClient(internal, baseURL)
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the scoring module to postgres: %w", err)
	}
	if err := db.RenameSchemaFrom(ctx, migrations.FormerSchema); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to rename the %s schema: %w", migrations.FormerSchema, err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the scoring module to redis: %w", err)
	}
	closeStores := func() {
		_ = streams.Close()
		_ = db.Close()
	}

	contributions := postgres_contribution_store.New(db)
	standingsQuery := standings_query.NewPostgresQuery(db, standings_authors.New(player), seasons, clock, countries)
	boards := inprocess_board_feed.New(log_board_reader.New(standingsQuery, props.Logger), clock)

	// The groups keep their names: a new name is a new group, which reads TileTaken again from the start of the stream.
	takes, err := cpredis.Subscribe(ctx, streams, "seasons-standings",
		log_subscriber.New(tile_taken_subscriber.New(record_take_usecase.New(seasons, contributions)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to planet.v1.TileTaken: %w", err)
	}
	deletions, err := cpredis.Subscribe(ctx, streams, "seasons-standings-accounts",
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(contributions)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}

	territories := rpc_planet_territories.New(planet)
	race := log_race_reader.New(
		race_query.NewPostgresQuery(db, territories, race_authors.New(player), seasons, clock), props.Logger)
	races := inprocess_race_feed.New(race)
	roundStore := postgres_round_store.New(db)
	outbox := cpoutbox.New(db)
	closing := close_ended_rounds_usecase.NewRunner(log_close_ended_rounds.New(
		close_ended_rounds_usecase.New(territories, roundStore,
			close_round_usecase.New(db, roundStore, outbox), close_season_usecase.New(db, roundStore, outbox), seasons, clock),
		props.Logger,
	), clock, close_ended_rounds_usecase.Retry)

	roundTakes, err := cpredis.Subscribe(ctx, streams, "seasons-rounds",
		log_subscriber.New(round_take_subscriber.New(count_take_usecase.New(roundStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the rounds to planet.v1.TileTaken: %w", err)
	}
	roundDeletions, err := cpredis.Subscribe(ctx, streams, "seasons-rounds-accounts",
		log_subscriber.New(account_deleted_subscriber.New(round_forget_account_usecase.New(roundStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the rounds to auth.v1.AccountDeleted: %w", err)
	}

	ruleSets := postgres_rule_set_store.New(db)
	play := play_rules_usecase.New(ruleSets, postgres_choice_store.New(db),
		open_own_round_usecase.New(seasons, roundStore, clock, closing))
	playRules := log_play_rules.New(play, props.Logger)
	if _, _, err := playRules.Execute(ctx); err != nil {
		closeStores()
		return fmt.Errorf("failed to play the season rules postgres keeps: %w", err)
	}
	chosen, err := cpredis.Handle(ctx, streams, play_rule_set_handler.New(playRules))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to handle scoring.v1.PlayRuleSet: %w", err)
	}

	props.Runners.Add(cppg.CloseAfter(db, props.Logger, boards, races, closing, cpredis.CloseAfter(streams, props.Logger,
		takes, deletions, roundTakes, roundDeletions, chosen, cpoutbox.NewRelay("scoring-outbox", db, streams, props.Logger))))

	service := scoringv1controller.SeasonService{
		GetSeasonHandler: get_season_handler.New(
			get_season_usecase.New(seasons, clock),
		),
		GetStandingsHandler: get_standings_handler.New(standingsQuery),
		GetMySeasonHandler: get_my_season_handler.New(my_season_query.NewPostgresQuery(
			db, my_season_authors.New(player), seasons, clock, countries,
		)),
		ListenForEventsHandler: listen_for_events_handler.New(boards, races, countries, props.Server.StreamHeartbeat),
		GetRaceHandler:         get_race_handler.New(race),
		GetHallOfFameHandler:   get_hall_of_fame_handler.New(hall_of_fame_query.NewPostgresQuery(db, hall_of_fame_authors.New(player))),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", moduleName)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return scoringv1connect.NewSeasonServiceHandler(service, options...)
	}, scoringv1controller.NewCacheInterceptor(),
		scoringv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
	); err != nil {
		return fmt.Errorf("failed to mount scoring.v1.SeasonService: %w", err)
	}

	admin := scoringv1controller.AdminService{
		GetRulesHandler: get_rules_handler.New(rule_sets_query.NewPostgresQuery(db), play),
		SaveRuleSetHandler: save_rule_set_handler.New(
			audit_save_rule_set.New(save_rule_set_usecase.New(ruleSets, play), props.Logger)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return scoringv1connect.NewAdminServiceHandler(admin, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount scoring.v1.AdminService: %w", err)
	}
	internalService := scoringv1controller.InternalService{
		GetRuleSetNamesHandler: get_rule_set_names_handler.New(rule_set_names_query.NewPostgresQuery(db)),
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return scoringv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount scoring.v1.InternalService: %w", err)
	}

	props.Logger.Info("scoring built", slog.Int("seasons", len(config.Calendar.List)), slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Calendar calendar.Config `koanf:",squash"`
	Database cppg.Config
	Redis    cpredis.Config
}

func (c Config) Validate() error {
	if err := c.Calendar.Validate(); err != nil {
		return fmt.Errorf("scoring: %w", err)
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("scoring.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("scoring.redis: %w", err)
	}
	return nil
}
