package clips

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/create_video_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler/appearances_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_appearance_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_snapshot_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_video_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_rows_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_rows_handler/rows_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_snapshots_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_snapshots_handler/snapshots_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_videos_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_videos_handler/videos_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/make_cast_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/remove_video_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/set_name_shown_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/postgres_optout_store"
	forget_optouts "github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/usecases/set_name_shown_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots/postgres_snapshot_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots/usecases/keep_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/postgres_video_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/s3_video_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/keep_video_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/remove_video_usecase"
	video_ids "github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
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
	config = config.withDefaults()

	files, err := cps3.New(config.Videos.bucket(), clock, &http.Client{Timeout: bucketTimeout})
	if err != nil {
		return fmt.Errorf("failed to build the bucket of the videos: %w", err)
	}
	videoBucket := s3_video_bucket.New(files, config.Videos.lifetimes())
	videoRules := videos.Rules{MaxBytes: config.Videos.MaxBytes}

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
	videoStore := postgres_video_store.New(db)
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
		CreateVideoUploadHandler: create_video_upload_handler.New(
			create_upload_usecase.New(video_ids.Provider{}, videoRules, videoBucket)),
		KeepVideoHandler:   keep_video_handler.New(keep_video_usecase.New(videoBucket, videoStore, videoRules, clock)),
		ListVideosHandler:  list_videos_handler.New(videos_query.NewPostgresQuery(db, videoBucket)),
		RemoveVideoHandler: remove_video_handler.New(remove_video_usecase.New(videoBucket, videoStore)),
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

	props.Logger.Info("clips built", slog.String("schema", config.Database.Schema), slog.Bool("videos", config.Videos.Enabled))
	return nil
}

// The server's settings, beside the job's Config: the job runs in a workflow, the module in the server.
type ModuleConfig struct {
	Database cppg.Config
	Redis    cpredis.Config
	// The videos the operator posts by hand, in the same bucket as the support module's files.
	Videos VideosConfig
}

type VideosConfig struct {
	Enabled  bool
	MaxBytes int64
	// The backoffice asks for a link right before it sends the file, and a big file takes a while on a slow line.
	UploadTTL time.Duration
	// A page left open on a phone still downloads.
	LinkTTL time.Duration
	Bucket  cps3.Config
}

// Off, the bucket block is not read, and every video call is FailedPrecondition.
func (c VideosConfig) bucket() cps3.Config {
	if !c.Enabled {
		return cps3.Config{}
	}
	return c.Bucket
}

func (c VideosConfig) lifetimes() s3_video_bucket.Lifetimes {
	return s3_video_bucket.Lifetimes{Upload: c.UploadTTL, View: c.LinkTTL}
}

const (
	bucketTimeout = 10 * time.Second

	defaultMaxVideoBytes = 500 << 20
	defaultUploadTTL     = 30 * time.Minute
	defaultLinkTTL       = 6 * time.Hour

	// A bucket refuses a link signed for longer.
	maxLinkTTL = 7 * 24 * time.Hour
)

func (c ModuleConfig) withDefaults() ModuleConfig {
	if c.Videos.MaxBytes <= 0 {
		c.Videos.MaxBytes = defaultMaxVideoBytes
	}
	if c.Videos.UploadTTL <= 0 {
		c.Videos.UploadTTL = defaultUploadTTL
	}
	if c.Videos.LinkTTL <= 0 {
		c.Videos.LinkTTL = defaultLinkTTL
	}
	return c
}

func (c ModuleConfig) Validate() error {
	var errs []error
	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("clips.database: %w", err))
	}
	if err := c.Redis.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("clips.redis: %w", err))
	}
	if !c.Videos.Enabled {
		return errors.Join(errs...)
	}
	if err := c.Videos.Bucket.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("clips.videos.bucket: %w", err))
	}
	for name, ttl := range map[string]time.Duration{"uploadTTL": c.Videos.UploadTTL, "linkTTL": c.Videos.LinkTTL} {
		if ttl > maxLinkTTL {
			errs = append(errs, fmt.Errorf("clips.videos.%s is %s, and a bucket takes %s at most", name, ttl, maxLinkTTL))
		}
	}
	return errors.Join(errs...)
}
