package metagame

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1/metagamev1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/add_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/delete_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/edit_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_schedule_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_schedule_handler/schedule_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_turn_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_turn_handler/turn_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/postgres_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/redis_rule_set_commands"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/rpc_antibot_rule_sets"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/rpc_planet_rule_sets"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/usecases/add_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/usecases/add_entry_usecase/audit_add_entry"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/usecases/delete_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/usecases/delete_entry_usecase/audit_delete_entry"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/usecases/edit_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/usecases/edit_entry_usecase/audit_edit_entry"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/usecases/play_schedule_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/usecases/play_schedule_usecase/log_play_schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	moduleName    = "metagame"
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
		return fmt.Errorf("the metagame module asks planet and the antibot which rule sets they keep: %w", err)
	}
	keepers := []schedule.Keeper{
		rpc_planet_rule_sets.New(planetv1connect.NewInternalServiceClient(internal, baseURL)),
		rpc_antibot_rule_sets.New(antibotv1connect.NewInternalServiceClient(internal, baseURL)),
	}

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the metagame module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the metagame module to redis: %w", err)
	}

	entries := postgres_entry_store.New(db)
	ruleSchedule := play_schedule_usecase.NewRunner(scheduleCheck, log_play_schedule.New(
		play_schedule_usecase.New(entries, redis_rule_set_commands.New(streams.Commands()), clock), props.Logger))
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger, ruleSchedule)))

	admin := metagamev1controller.AdminService{
		GetScheduleHandler: get_schedule_handler.New(schedule_query.NewPostgresQuery(db, clock)),
		AddEntryHandler: add_entry_handler.New(audit_add_entry.New(
			add_entry_usecase.New(entries, keepers, uuid_id_provider.Provider{}, clock), props.Logger)),
		EditEntryHandler: edit_entry_handler.New(audit_edit_entry.New(
			edit_entry_usecase.New(entries, keepers, clock), props.Logger)),
		DeleteEntryHandler: delete_entry_handler.New(audit_delete_entry.New(
			delete_entry_usecase.New(entries, clock), props.Logger)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return metagamev1connect.NewAdminServiceHandler(admin, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount metagame.v1.AdminService: %w", err)
	}

	public := metagamev1controller.MetagameService{
		GetTurnHandler: get_turn_handler.New(turn_query.NewPostgresQuery(db, clock)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return metagamev1connect.NewMetagameServiceHandler(public, options...)
	}, metagamev1controller.NewCacheInterceptor()); err != nil {
		return fmt.Errorf("failed to mount metagame.v1.MetagameService: %w", err)
	}

	props.Logger.Info("metagame built", slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Database cppg.Config
	Redis    cpredis.Config
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("metagame.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("metagame.redis: %w", err)
	}
	return nil
}
