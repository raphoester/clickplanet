package clips

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1/clipsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances/postgres_appearance_store"
	forget_appearances "github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances/usecases/keep_appearance_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/postgres_cast_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/rpc_planet_ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/rpc_player_names"
	forget_casts "github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/usecases/make_cast_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler/appearances_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_appearance_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_snapshot_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_rows_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_rows_handler/rows_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_snapshots_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_snapshots_handler/snapshots_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/make_cast_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/set_name_shown_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/postgres_optout_store"
	forget_optouts "github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/usecases/set_name_shown_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots/postgres_snapshot_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots/usecases/keep_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "clips"

// The server's side of the clips job: the numbers of the clips it posted, a snapshot a day, which `clips stats` sends
// over ssh and the backoffice reads; the cast of each run and who each post names, which `clips fetch` and `clips
// schedule` ask over ssh; and, on the public router, the posts that name the caller.
func NewModule(config ModuleConfig) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config, props)
		},
	}
}

func build(ctx context.Context, config ModuleConfig, props cpbootstrap.Props) error {
	clock := cptime.SystemClock{}

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the clips module asks planet who took what and player for their names: %w", err)
	}

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the clips module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the clips module to redis: %w", err)
	}
	closeStores := func() {
		_ = streams.Close()
		_ = db.Close()
	}

	castStore := postgres_cast_store.New(db)
	appearances := postgres_appearance_store.New(db)
	optOuts := postgres_optout_store.New(db)

	forgottenCasts, err := cpredis.Subscribe(ctx, streams, "clips-casts-accounts",
		log_subscriber.New(account_deleted_subscriber.New(forget_casts.New(castStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the casts to auth.v1.AccountDeleted: %w", err)
	}
	forgottenAppearances, err := cpredis.Subscribe(ctx, streams, "clips-appearances-accounts",
		log_subscriber.New(account_deleted_subscriber.New(forget_appearances.New(appearances)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the appearances to auth.v1.AccountDeleted: %w", err)
	}
	forgottenOptOuts, err := cpredis.Subscribe(ctx, streams, "clips-optouts-accounts",
		log_subscriber.New(account_deleted_subscriber.New(forget_optouts.New(optOuts)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the opt-outs to auth.v1.AccountDeleted: %w", err)
	}
	// Not closers: closers run before the runners stop, and the runners use both.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger,
		forgottenCasts, forgottenAppearances, forgottenOptOuts)))

	adminService := clipsv1controller.AdminService{
		KeepSnapshotHandler:  keep_snapshot_handler.New(keep_snapshot_usecase.New(postgres_snapshot_store.New(db))),
		ListSnapshotsHandler: list_snapshots_handler.New(snapshots_query.NewPostgresQuery(db)),
		ListRowsHandler:      list_rows_handler.New(rows_query.NewPostgresQuery(db)),
		MakeCastHandler: make_cast_handler.New(make_cast_usecase.New(
			rpc_planet_ledger.New(planetv1connect.NewInternalServiceClient(internal, baseURL)),
			casts.NewShownNames(rpc_player_names.New(playerv1connect.NewInternalServiceClient(internal, baseURL)), optOuts),
			castStore, uuid_id_provider.Provider{}, clock,
		)),
		KeepAppearanceHandler: keep_appearance_handler.New(keep_appearance_usecase.New(castStore, appearances, clock)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return clipsv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount clips.v1.AdminService: %w", err)
	}

	clipsService := clipsv1controller.ClipsService{
		GetAppearancesHandler: get_appearances_handler.New(appearances_query.NewPostgresQuery(db, clock)),
		SetNameShownHandler:   set_name_shown_handler.New(set_name_shown_usecase.New(optOuts)),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", moduleName)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return clipsv1connect.NewClipsServiceHandler(clipsService, options...)
	}, clipsv1controller.NewSessionInterceptor(verifier, clock, props.Metrics)); err != nil {
		return fmt.Errorf("failed to mount clips.v1.ClipsService: %w", err)
	}

	props.Logger.Info("clips built", slog.String("schema", config.Database.Schema))
	return nil
}

// The server's settings, beside the job's Config: the job runs in a workflow, the module in the server.
type ModuleConfig struct {
	Database cppg.Config
	Redis    cpredis.Config
}

func (c ModuleConfig) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("clips.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("clips.redis: %w", err)
	}
	return nil
}
