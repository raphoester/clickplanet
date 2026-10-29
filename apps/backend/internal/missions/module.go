package missions

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1/missionsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days/postgres_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days/usecases/keep_day_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/payroll"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/postgres_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/redis_outbox_sender"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/rpc_auth_accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/rpc_planet_allegiance"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/rpc_planet_world"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/pay_signed_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/record_fact_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_missions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/bomb_landed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/bonus_claimed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/day_started_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/fortified_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/quiz_answered_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/shield_placed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/sign_in_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "missions"

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
		return fmt.Errorf("the missions module asks planet about the map and auth about accounts: %w", err)
	}
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)
	auth := authv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the missions module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the missions module to redis: %w", err)
	}
	closeStores := func() {
		_ = streams.Close()
		_ = db.Close()
	}

	calendar := postgres_day_store.New(db)
	store := postgres_mission_store.New(db)
	briefing := missions.NewBriefing(store, rpc_planet_world.New(planet), rpc_planet_allegiance.New(planet))
	pay := payroll.New(store, rpc_auth_accounts.New(auth), cpoutbox.New(db))

	told, err := cpredis.Subscribe(ctx, streams, "missions-days",
		log_subscriber.New(day_started_subscriber.New(keep_day_usecase.New(calendar)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to gamemaster.v1.DayStarted: %w", err)
	}
	facts, err := subscribeFacts(ctx, streams, record_fact_usecase.New(calendar, briefing, store, db, pay), props.Logger)
	if err != nil {
		closeStores()
		return err
	}
	deletions, err := cpredis.Subscribe(ctx, streams, "missions-accounts",
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(store)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}
	signIns, err := cpredis.Subscribe(ctx, streams, "missions-sign-ins",
		log_subscriber.New(sign_in_subscriber.New(pay_signed_in_usecase.New(calendar, clock, db, pay)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to auth.v1.SignedIn: %w", err)
	}

	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger,
		append(facts, told, deletions, signIns, cpoutbox.NewRelay("missions-outbox", db, redis_outbox_sender.New(streams, streams.Commands()), props.Logger))...)))

	service := missionsv1controller.MissionService{
		GetMissionsHandler: get_missions_handler.New(
			get_missions_usecase.New(calendar, clock, store, briefing),
		),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", moduleName)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return missionsv1connect.NewMissionServiceHandler(service, options...)
	}, missionsv1controller.NewSessionInterceptor(verifier, clock, props.Metrics)); err != nil {
		return fmt.Errorf("failed to mount missions.v1.MissionService: %w", err)
	}

	props.Logger.Info("missions built", slog.String("schema", config.Database.Schema))

	return nil
}

// Each kind of fact a mission counts, on its own stream. A group keeps its name: a new one reads its stream again.
func subscribeFacts(
	ctx context.Context, streams *cpredis.Streams, record *record_fact_usecase.UseCase, logger *slog.Logger,
) ([]cpbootstrap.Runner, error) {
	takes, err := cpredis.Subscribe(ctx, streams, "missions-progress", log_subscriber.New(tile_taken_subscriber.New(record), logger))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to planet.v1.TileTaken: %w", err)
	}
	boxes, err := cpredis.Subscribe(ctx, streams, "missions-boxes", log_subscriber.New(bonus_claimed_subscriber.New(record), logger))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to planet.v1.BonusClaimed: %w", err)
	}
	answers, err := cpredis.Subscribe(ctx, streams, "missions-answers", log_subscriber.New(quiz_answered_subscriber.New(record), logger))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to planet.v1.QuizAnswered: %w", err)
	}
	bombs, err := cpredis.Subscribe(ctx, streams, "missions-bombs", log_subscriber.New(bomb_landed_subscriber.New(record), logger))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to planet.v1.BombLanded: %w", err)
	}
	shields, err := cpredis.Subscribe(ctx, streams, "missions-shields", log_subscriber.New(shield_placed_subscriber.New(record), logger))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to planet.v1.ShieldPlaced: %w", err)
	}
	fortifies, err := cpredis.Subscribe(ctx, streams, "missions-fortifies", log_subscriber.New(fortified_subscriber.New(record), logger))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to planet.v1.Fortified: %w", err)
	}
	return []cpbootstrap.Runner{takes, boxes, answers, bombs, shields, fortifies}, nil
}

type Config struct {
	Database cppg.Config
	Redis    cpredis.Config
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("missions.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("missions.redis: %w", err)
	}
	return nil
}
