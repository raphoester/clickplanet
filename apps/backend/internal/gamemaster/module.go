package gamemaster

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/add_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/delete_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/edit_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_schedule_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_schedule_handler/schedule_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_turn_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_turn_handler/turn_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/listen_for_events_handler/inprocess_turn_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/listen_for_events_handler/inprocess_turn_feed/log_turn_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/postgres_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/redis_rule_set_commands"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/rpc_antibot_rule_sets"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/rpc_planet_rule_sets"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/rpc_scoring_rule_sets"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/add_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/add_entry_usecase/audit_add_entry"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/delete_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/delete_entry_usecase/audit_delete_entry"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/edit_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/edit_entry_usecase/audit_edit_entry"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/play_schedule_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/play_schedule_usecase/log_play_schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	moduleName    = "gamemaster"
	scheduleCheck = time.Second
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

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the gamemaster module asks planet, the antibot and scoring which rule sets they keep: %w", err)
	}
	keepers := []schedule.Keeper{
		rpc_planet_rule_sets.New(planetv1connect.NewInternalServiceClient(internal, baseURL)),
		rpc_antibot_rule_sets.New(antibotv1connect.NewInternalServiceClient(internal, baseURL)),
		rpc_scoring_rule_sets.New(scoringv1connect.NewInternalServiceClient(internal, baseURL)),
	}

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the gamemaster module to postgres: %w", err)
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
		return fmt.Errorf("failed to connect the gamemaster module to redis: %w", err)
	}

	entries := postgres_entry_store.New(db)
	ruleSchedule := play_schedule_usecase.NewRunner(scheduleCheck, log_play_schedule.New(
		play_schedule_usecase.New(entries, redis_rule_set_commands.New(streams.Commands()), clock), props.Logger))
	turns := turn_query.NewPostgresQuery(db, clock)
	turnFeed := inprocess_turn_feed.New(log_turn_reader.New(turns, props.Logger))
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, turnFeed, cpredis.CloseAfter(streams, props.Logger, ruleSchedule)))

	admin := gamemasterv1controller.AdminService{
		GetScheduleHandler: get_schedule_handler.New(schedule_query.NewPostgresQuery(db, clock)),
		AddEntryHandler: add_entry_handler.New(audit_add_entry.New(
			add_entry_usecase.New(entries, keepers, uuid_id_provider.Provider{}, clock), props.Logger)),
		EditEntryHandler: edit_entry_handler.New(audit_edit_entry.New(
			edit_entry_usecase.New(entries, keepers, clock), props.Logger)),
		DeleteEntryHandler: delete_entry_handler.New(audit_delete_entry.New(
			delete_entry_usecase.New(entries, clock), props.Logger)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return gamemasterv1connect.NewAdminServiceHandler(admin, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount gamemaster.v1.AdminService: %w", err)
	}

	public := gamemasterv1controller.GamemasterService{
		GetTurnHandler:         get_turn_handler.New(turns),
		ListenForEventsHandler: listen_for_events_handler.New(turnFeed, props.Server.StreamHeartbeat),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return gamemasterv1connect.NewGamemasterServiceHandler(public, options...)
	}, gamemasterv1controller.NewCacheInterceptor()); err != nil {
		return fmt.Errorf("failed to mount gamemaster.v1.GamemasterService: %w", err)
	}

	props.Logger.Info("gamemaster built", slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Database cppg.Config
	Redis    cpredis.Config
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("gamemaster.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("gamemaster.redis: %w", err)
	}
	return nil
}
