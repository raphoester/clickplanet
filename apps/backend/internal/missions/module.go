package missions

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1/missionsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/postgres_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/rpc_planet_charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/rpc_planet_world"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/collect_mission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/collect_mission_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_missions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "missions"

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
	calendar := missions.NewCalendar(config.Calendar)

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the missions module asks planet about the map and for rewards: %w", err)
	}
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the missions module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	store := postgres_mission_store.New(db)
	briefing := missions.NewBriefing(store, rpc_planet_world.New(planet))

	takes, err := cpbootstrap.Subscribe(props.Events, "missions-progress", tileTakenBuffer,
		log_subscriber.New(tile_taken_subscriber.New(record_take_usecase.New(calendar, briefing, store)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to planet.v1.TileTaken: %w", err)
	}
	deletions, err := cpbootstrap.Subscribe(props.Events, "missions-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(store)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}

	props.Runners.Add(cppg.CloseAfter(db, props.Logger, takes, deletions))

	service := missionsv1controller.MissionService{
		GetMissionsHandler: get_missions_handler.New(
			get_missions_usecase.New(calendar, clock, store, briefing, cpcountries.New()),
		),
		CollectMissionHandler: collect_mission_handler.New(
			collect_mission_usecase.New(calendar, clock, store, briefing, rpc_planet_charges.New(planet)),
		),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", moduleName)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return missionsv1connect.NewMissionServiceHandler(service, options...)
	}, missionsv1controller.NewSessionInterceptor(verifier, clock, props.Metrics)); err != nil {
		return fmt.Errorf("failed to mount missions.v1.MissionService: %w", err)
	}

	props.Logger.Info("missions built", slog.Duration("cutoff", config.Calendar.Cutoff),
		slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Calendar missions.Config `koanf:",squash"`
	Database cppg.Config
}

func (c Config) Validate() error {
	if err := c.Calendar.Validate(); err != nil {
		return fmt.Errorf("missions: %w", err)
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("missions.database: %w", err)
	}
	return nil
}
