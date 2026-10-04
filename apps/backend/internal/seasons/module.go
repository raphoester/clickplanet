package seasons

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/get_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/rpc_player_names"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/get_my_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/get_standings_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/record_take_usecase"
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

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the seasons module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	contributions := postgres_contribution_store.New(db)
	board := standings.NewBoard(contributions, rpc_player_names.New(props.Internal))

	takes, err := cpbootstrap.Subscribe(props.Events, "seasons-standings", tileTakenBuffer,
		log_subscriber.New(tile_taken_subscriber.New(record_take_usecase.New(seasons, contributions)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to planet.v1.TileTaken: %w", err)
	}
	deletions, err := cpbootstrap.Subscribe(props.Events, "seasons-standings-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(contributions)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, takes, deletions))

	service := seasonsv1controller.SeasonService{
		GetSeasonHandler: get_season_handler.New(
			get_season_usecase.New(seasons, clock),
		),
		GetStandingsHandler: get_standings_handler.New(
			get_standings_usecase.New(seasons, clock, cpcountries.New(), board),
		),
		GetMySeasonHandler: get_my_season_handler.New(get_my_season_usecase.New(seasons, clock, board)),
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
