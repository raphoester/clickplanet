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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler/my_season_query"
	my_season_authors "github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler/my_season_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler/standings_query"
	standings_authors "github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler/standings_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/rebuild_standings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/rpc_take_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/count_takes_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/count_takes_usecase/log_count_takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/rebuild_standings_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/rebuild_standings_usecase/audit_rebuild_standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "seasons"

const accountDeletedBuffer = 2048

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

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the seasons module asks the player module who plays and reads planet's log: %w", err)
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

	counting := count_takes_usecase.NewRunner(config.Takes, log_count_takes.New(
		count_takes_usecase.New(rpc_take_feed.New(planet), contributions, seasons), props.Logger))
	deletions, err := cpbootstrap.Subscribe(props.Events, "seasons-standings-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(contributions)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, counting, deletions))

	service := seasonsv1controller.SeasonService{
		GetSeasonHandler: get_season_handler.New(
			get_season_usecase.New(seasons, clock),
		),
		GetStandingsHandler: get_standings_handler.New(standings_query.NewPostgresQuery(
			db, standings_authors.New(player), seasons, clock, cpcountries.New(),
		)),
		GetMySeasonHandler: get_my_season_handler.New(my_season_query.NewPostgresQuery(
			db, my_season_authors.New(player), seasons, clock,
		)),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", "seasons")))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return seasonsv1connect.NewSeasonServiceHandler(service, options...)
	}, seasonsv1controller.NewCacheInterceptor(),
		seasonsv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
	); err != nil {
		return fmt.Errorf("failed to mount seasons.v1.SeasonService: %w", err)
	}

	adminService := seasonsv1controller.AdminService{
		RebuildStandingsHandler: rebuild_standings_handler.New(audit_rebuild_standings.New(
			rebuild_standings_usecase.New(contributions), props.Logger,
		)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return seasonsv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount seasons.v1.AdminService: %w", err)
	}

	props.Logger.Info("seasons built", slog.Int("seasons", len(config.Calendar.List)), slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Calendar calendar.Config `koanf:",squash"`
	Database cppg.Config
	Takes    count_takes_usecase.Config
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
