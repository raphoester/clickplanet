package seasons

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/get_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/rpc_planet_territories"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/take_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/take_snapshot_usecase/log_take_snapshot"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler/my_season_query"
	my_season_authors "github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler/my_season_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler/standings_query"
	standings_authors "github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler/standings_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/inprocess_board_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/inprocess_board_feed/log_board_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/inprocess_race_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/inprocess_race_feed/log_race_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/race_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/forget_account_usecase/marking_forget_account"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/record_take_usecase/marking_record_take"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "seasons"

const (
	tileTakenBuffer      = 8192
	accountDeletedBuffer = 2048
)

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
		return fmt.Errorf("the seasons module asks the player module who plays: %w", err)
	}
	player := playerv1connect.NewInternalServiceClient(internal, baseURL)
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the seasons module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	contributions := postgres_contribution_store.New(db)
	standingsQuery := standings_query.NewPostgresQuery(db, standings_authors.New(player), seasons, clock, countries)
	boards := inprocess_board_feed.New(log_board_reader.New(standingsQuery, props.Logger), clock)

	takes, err := cpbootstrap.Subscribe(props.Events, "seasons-standings", tileTakenBuffer,
		log_subscriber.New(tile_taken_subscriber.New(
			marking_record_take.New(record_take_usecase.New(seasons, contributions), boards),
		), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to planet.v1.TileTaken: %w", err)
	}
	deletions, err := cpbootstrap.Subscribe(props.Events, "seasons-standings-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(
			marking_forget_account.New(forget_account_usecase.New(contributions), boards),
		), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}

	races := inprocess_race_feed.New(log_race_reader.New(race_query.NewPostgresQuery(db, seasons, clock), props.Logger))
	snapshot := take_snapshot_usecase.NewRunner(config.Snapshot, log_take_snapshot.New(
		take_snapshot_usecase.New(rpc_planet_territories.New(planet), postgres_round_store.New(db), seasons, clock),
		props.Logger,
	))

	props.Runners.Add(cppg.CloseAfter(db, props.Logger, takes, deletions, boards, races, snapshot))

	service := seasonsv1controller.SeasonService{
		GetSeasonHandler: get_season_handler.New(
			get_season_usecase.New(seasons, clock),
		),
		GetStandingsHandler: get_standings_handler.New(standingsQuery),
		GetMySeasonHandler: get_my_season_handler.New(my_season_query.NewPostgresQuery(
			db, my_season_authors.New(player), seasons, clock, countries,
		)),
		ListenForEventsHandler: listen_for_events_handler.New(boards, races, countries, props.Server.StreamHeartbeat),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", "seasons")))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return seasonsv1connect.NewSeasonServiceHandler(service, options...)
	}, seasonsv1controller.NewCacheInterceptor(),
		seasonsv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
	); err != nil {
		return fmt.Errorf("failed to mount seasons.v1.SeasonService: %w", err)
	}

	props.Logger.Info("seasons built", slog.Int("seasons", len(config.Calendar.List)), slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Calendar calendar.Config `koanf:",squash"`
	Snapshot take_snapshot_usecase.Config
	Database cppg.Config
}

func (c Config) Validate() error {
	if err := c.Calendar.Validate(); err != nil {
		return fmt.Errorf("seasons: %w", err)
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("seasons.database: %w", err)
	}
	return nil
}
