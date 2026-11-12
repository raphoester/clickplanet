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
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"

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
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler/appearances_query/rpc_publishing_links"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_shown_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_shown_names_handler/shown_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_appearance_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_video_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_name_verdicts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_name_verdicts_handler/verdicts_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_videos_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_videos_handler/videos_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/make_cast_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/remove_video_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/set_name_shown_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/set_name_verdict_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/postgres_optout_store"
	forget_optouts "github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/usecases/set_name_shown_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/subscribers/player_forgotten_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/subscribers/player_named_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/claude_name_judge"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/postgres_verdict_store"
	forget_verdicts "github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/judge_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/screen_names_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/set_verdict_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/postgres_video_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/s3_video_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/keep_video_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/remove_video_usecase"
	video_ids "github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "clips"

// The server's side of the clips job: the cast of each run and who each post names, which `clips fetch` and `clips
// schedule` ask over ssh; the videos the operator posts by hand; and, on the public router, the posts that name the
// caller. The numbers of the posts are the publishing module's.
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
	nameJudge, err := nameJudgeOf(config.Judge)
	if err != nil {
		return err
	}
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
	nameVerdicts := postgres_verdict_store.New(db)
	usernames := rpc_player_names.New(playerv1connect.NewInternalServiceClient(internal, baseURL))
	screening := screen_names_usecase.New(nameVerdicts, nameJudge, clock)

	forgottenCasts, err := cpredis.Subscribe(ctx, streams, "clips-casts-forgotten",
		log_subscriber.New(player_forgotten_subscriber.New(forget_casts.New(castStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the casts to player.v1.PlayerForgotten: %w", err)
	}
	forgottenAppearances, err := cpredis.Subscribe(ctx, streams, "clips-appearances-forgotten",
		log_subscriber.New(player_forgotten_subscriber.New(forget_appearances.New(appearances)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the appearances to player.v1.PlayerForgotten: %w", err)
	}
	forgottenOptOuts, err := cpredis.Subscribe(ctx, streams, "clips-optouts-forgotten",
		log_subscriber.New(player_forgotten_subscriber.New(forget_optouts.New(optOuts)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the opt-outs to player.v1.PlayerForgotten: %w", err)
	}
	forgottenVerdicts, err := cpredis.Subscribe(ctx, streams, "clips-verdicts-forgotten",
		log_subscriber.New(player_forgotten_subscriber.New(forget_verdicts.New(nameVerdicts)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the name verdicts to player.v1.PlayerForgotten: %w", err)
	}
	subscriptions := []cpbootstrap.Runner{forgottenCasts, forgottenAppearances, forgottenOptOuts, forgottenVerdicts}
	if config.Judge.keyed() {
		namedVerdicts, err := cpredis.Subscribe(ctx, streams, "clips-verdicts-named",
			log_subscriber.New(player_named_subscriber.New(judge_name_usecase.New(usernames, screening)), props.Logger))
		if err != nil {
			closeStores()
			return fmt.Errorf("failed to subscribe the name verdicts to player.v1.PlayerNamed: %w", err)
		}
		subscriptions = append(subscriptions, namedVerdicts)
	}
	// Not closers: closers run before the runners stop, and the runners use both.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger, subscriptions...)))

	adminService := clipsv1controller.AdminService{
		MakeCastHandler: make_cast_handler.New(make_cast_usecase.New(
			rpc_planet_ledger.New(planetv1connect.NewInternalServiceClient(internal, baseURL)),
			casts.NewShownNames(usernames, optOuts, screening),
			castStore, uuid_id_provider.Provider{}, clock,
		)),
		KeepAppearanceHandler: keep_appearance_handler.New(keep_appearance_usecase.New(castStore, appearances, clock)),
		CreateVideoUploadHandler: create_video_upload_handler.New(
			create_upload_usecase.New(video_ids.Provider{}, videoRules, videoBucket)),
		KeepVideoHandler:        keep_video_handler.New(keep_video_usecase.New(videoBucket, videoStore, videoRules, clock)),
		ListVideosHandler:       list_videos_handler.New(videos_query.NewPostgresQuery(db, videoBucket)),
		RemoveVideoHandler:      remove_video_handler.New(remove_video_usecase.New(videoBucket, videoStore)),
		ListNameVerdictsHandler: list_name_verdicts_handler.New(verdicts_query.NewPostgresQuery(db)),
		SetNameVerdictHandler:   set_name_verdict_handler.New(set_verdict_usecase.New(nameVerdicts, clock)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return clipsv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount clips.v1.AdminService: %w", err)
	}

	clipsService := clipsv1controller.ClipsService{
		GetAppearancesHandler: get_appearances_handler.New(appearances_query.NewPostgresQuery(db,
			rpc_publishing_links.New(publishingv1connect.NewInternalServiceClient(internal, baseURL), props.Logger), clock)),
		SetNameShownHandler:  set_name_shown_handler.New(set_name_shown_usecase.New(optOuts)),
		GetShownNamesHandler: get_shown_names_handler.New(shown_names_query.NewPostgresQuery(db, usernames)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return clipsv1connect.NewClipsServiceHandler(clipsService, options...)
	}, clipsv1controller.NewSessionInterceptor(props.Sessions, clock, props.Metrics)); err != nil {
		return fmt.Errorf("failed to mount clips.v1.ClipsService: %w", err)
	}

	props.Logger.Info("clips built", slog.String("schema", config.Database.Schema), slog.Bool("videos", config.Videos.Enabled),
		slog.Bool("nameJudge", config.Judge.keyed()))
	return nil
}

// The server's settings, beside the job's Config: the job runs in a workflow, the module in the server.
type ModuleConfig struct {
	Database cppg.Config
	Redis    cpredis.Config
	// The videos the operator posts by hand, in the same bucket as the support module's files.
	Videos VideosConfig
	Judge  NameJudgeConfig
}

type NameJudgeConfig struct {
	// ANTHROPIC_API_KEY. None sends nothing, and no cast names anyone.
	Anthropic cpanthropic.Config
	// Unset is claude_name_judge.DefaultModel, Claude Haiku 5.5.
	Model string
}

// Unkeyed, no name is judged: the module does not hear the names set.
func (c NameJudgeConfig) keyed() bool {
	return c.Anthropic.APIKey != ""
}

// The judge of the names on a client of Anthropic's API built here; with no key, one that sends nothing.
func nameJudgeOf(config NameJudgeConfig) (screen_names_usecase.Judge, error) {
	client, err := cpanthropic.NewClient(config.Anthropic)
	if errors.Is(err, cpanthropic.ErrNoKey) {
		return claude_name_judge.Unkeyed{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build the name judge's client of Anthropic's API: %w", err)
	}
	return claude_name_judge.New(client, config.Model), nil
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
	if c.Judge.Model == "" {
		c.Judge.Model = claude_name_judge.DefaultModel
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
