package flags

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1/flagsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/random_flag_ids"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/add_time_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/close_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/expire_flags_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/open_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/restate_flags_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/tell_streams_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/add_time_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/close_flag_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/create_flag_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_flags_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_flags_handler/flags_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/open_flag_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures/s3_picture_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures/uuid_upload_ids"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/subscribers/told_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "flags"

// The flags a page that reads nothing may fall behind before its stream is closed: it opens it again and reads the list.
const streamBuffer = 32

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config.withDefaults(), props)
		},
	}
}

func build(ctx context.Context, config Config, props cpbootstrap.Props) error {
	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the flags module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	redis := cpredis.New(config.Redis, props.Metrics)
	if err := redis.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the flags module to redis: %w", err)
	}

	clock := cptime.SystemClock{}
	files, err := cps3.New(config.Pictures.bucket(), clock, &http.Client{Timeout: bucketTimeout})
	if err != nil {
		_ = redis.Close()
		_ = db.Close()
		return fmt.Errorf("failed to build the bucket of the flags' pictures: %w", err)
	}
	bucket := s3_picture_bucket.New(files, config.Pictures.UploadTTL)
	rules := pictures.Rules{MaxBytes: config.Pictures.MaxBytes}
	messages := flagmessage.New(config.Pictures.PublicBaseURL)

	store := postgres_flag_store.New(db)
	// Facts decided before this boot are told again for the other modules, never to the pages: they read the list.
	streams := feed.NewBoard(streamBuffer)
	tell := tell_streams_usecase.New(feed.NewsSince(clock.Now()), store, streams)
	opened, err := cpredis.Subscribe(ctx, redis, "flags-streams-opened",
		log_subscriber.New(told_subscriber.NewOpened(tell), props.Logger))
	if err != nil {
		_ = redis.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the streams to flags.v1.FlagOpened: %w", err)
	}
	closed, err := cpredis.Subscribe(ctx, redis, "flags-streams-closed",
		log_subscriber.New(told_subscriber.NewClosed(tell), props.Logger))
	if err != nil {
		_ = redis.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the streams to flags.v1.FlagClosed: %w", err)
	}

	// Each module keeps what it heard: one that subscribed after a flag was told hears it at the next boot.
	if err := restate_flags_usecase.New(db, store, cpoutbox.New(db)).Execute(ctx); err != nil {
		_ = redis.Close()
		_ = db.Close()
		return err //nolint:wrapcheck // the use case named it.
	}

	// Not a closer: closers run before the runners stop, and the relay uses the pool.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger,
		expire_flags_usecase.NewRunner(expiryInterval, expire_flags_usecase.New(clock, db, store, cpoutbox.New(db)),
			props.Logger),
		cpredis.CloseAfter(redis, props.Logger, opened, closed, cpoutbox.NewRelay("flags-outbox", db, redis, props.Logger))))

	list := list_flags_handler.New(flags_query.NewPostgresQuery(db, messages))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return flagsv1connect.NewFlagServiceHandler(flagsv1controller.FlagService{
			ListFlagsHandler: list,
			ListenForEventsHandler: listen_for_events_handler.New(
				listen_for_events_usecase.New(streams, props.Server.StreamHeartbeat), messages),
		}, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount flags.v1.FlagService: %w", err)
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return flagsv1connect.NewAdminServiceHandler(flagsv1controller.AdminService{
			ListFlagsHandler: list,
			CreateFlagUploadHandler: create_flag_upload_handler.New(
				create_upload_usecase.New(uuid_upload_ids.Provider{}, rules, bucket)),
			OpenFlagHandler: open_flag_handler.New(open_flag_usecase.New(
				catalog.NewOpening(random_flag_ids.Provider{}, clock), pictures.NewGallery(bucket, rules), db, store,
				cpoutbox.New(db)), messages),
			AddTimeHandler:   add_time_handler.New(add_time_usecase.New(clock, db, store, cpoutbox.New(db)), messages),
			CloseFlagHandler: close_flag_handler.New(close_flag_usecase.New(clock, db, store, cpoutbox.New(db)), messages),
		}, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount flags.v1.AdminService: %w", err)
	}

	props.Logger.Info("flags built", slog.String("schema", config.Database.Schema),
		slog.Bool("pictures", config.Pictures.Enabled))
	return nil
}

type Config struct {
	// The flags opened since the game shipped, and the outbox of what is told of them.
	Database cppg.Config
	// Where it tells each flag opened or closed.
	Redis cpredis.Config
	// The pictures of the flags opened since the game shipped.
	Pictures PicturesConfig
}

// In the support module's bucket: uploaded under flags/uploads/, private, and drawn again under flags/pictures/, which
// Cloudflare serves on the public domain.
type PicturesConfig struct {
	Enabled  bool
	MaxBytes int64
	// The backoffice asks for a link right before it sends the file.
	UploadTTL time.Duration
	// The public domain's origin, such as https://media.clickplanet.lol: the pages read the pictures from it.
	PublicBaseURL string
	Bucket        cps3.Config
}

// Off, the bucket block is not read, and every picture call is FailedPrecondition: no flag can be opened.
func (c PicturesConfig) bucket() cps3.Config {
	if !c.Enabled {
		return cps3.Config{}
	}
	return c.Bucket
}

const (
	bucketTimeout = 10 * time.Second
	// A flag closes at most this long after its end.
	expiryInterval = 30 * time.Second

	defaultMaxPictureBytes = 5 << 20
	defaultUploadTTL       = 10 * time.Minute
	// A bucket refuses a link signed for longer.
	maxLinkTTL = 7 * 24 * time.Hour
)

func (c Config) withDefaults() Config {
	if c.Pictures.MaxBytes <= 0 {
		c.Pictures.MaxBytes = defaultMaxPictureBytes
	}
	if c.Pictures.UploadTTL <= 0 {
		c.Pictures.UploadTTL = defaultUploadTTL
	}
	return c
}

func (c Config) Validate() error {
	var errs []error
	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("flags.database: %w", err))
	}
	if err := c.Redis.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("flags.redis: %w", err))
	}
	if !c.Pictures.Enabled {
		return errors.Join(errs...)
	}
	if err := c.Pictures.Bucket.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("flags.pictures.bucket: %w", err))
	}
	if base, err := url.Parse(c.Pictures.PublicBaseURL); err != nil || base.Scheme != "https" || base.Host == "" ||
		(base.Path != "" && base.Path != "/") || base.RawQuery != "" {
		errs = append(errs, fmt.Errorf("flags.pictures.publicBaseURL is %q, and must be an https origin such as "+
			"https://media.clickplanet.lol", c.Pictures.PublicBaseURL))
	}
	if c.Pictures.UploadTTL > maxLinkTTL {
		errs = append(errs, fmt.Errorf("flags.pictures.uploadTTL is %s, and a bucket takes %s at most",
			c.Pictures.UploadTTL, maxLinkTTL))
	}
	return errors.Join(errs...)
}
