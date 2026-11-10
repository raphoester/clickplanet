// The game's posts on its own TikTok and YouTube: when each goes out, within what each platform takes in a day, through
// the one posting service, Zernio. The clips job asks it over ssh, on the console listener; the promo module, on the
// internal listener.
package publishing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/remembered_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/usecases/schedule_post_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/usecases/schedule_post_usecase/log_schedule_post"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/zernio"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/create_media_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/get_post_links_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/get_post_links_handler/links_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/get_room_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/get_room_handler/room_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_posts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_rows_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_rows_handler/rows_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_snapshots_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_snapshots_handler/snapshots_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/plan_post_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/plan_post_handler/plan_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/schedule_post_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats/postgres_snapshot_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats/usecases/take_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats/usecases/take_snapshot_usecase/log_take_snapshot"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats/zernio_source"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "publishing"

// Each call to the posting service: its calls carry a little JSON, never a video.
const serviceTimeout = time.Minute

// How long a busy service is waited for, during the snapshot: as the clips job's sender waited.
var snapshotBackoff = stats.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config, props)
		},
	}
}

// What the operator calls and what another module calls: the same handlers.
type services interface {
	publishingv1connect.AdminServiceHandler
	publishingv1connect.InternalServiceHandler
}

func build(ctx context.Context, config Config, props cpbootstrap.Props) error {
	service, err := serviceOf(ctx, config, props)
	if err != nil {
		return err
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return publishingv1connect.NewAdminServiceHandler(service, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount publishing.v1.AdminService: %w", err)
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return publishingv1connect.NewInternalServiceHandler(service, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount publishing.v1.InternalService: %w", err)
	}
	return nil
}

// Off with no key: every call says so, and the server still starts.
func serviceOf(ctx context.Context, config Config, props cpbootstrap.Props) (services, error) {
	if config.Zernio.APIKey == "" {
		props.Logger.Warn("publishing is off: no ZERNIO_API_KEY")
		return publishingv1controller.Off{}, nil
	}

	zernioService, err := zernio.New(zernio.Config{APIKey: config.Zernio.APIKey, BaseURL: config.Zernio.BaseURL},
		&http.Client{Timeout: serviceTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to build the posting service: %w", err)
	}
	clock := cptime.SystemClock{}
	// One for every read and every post: a post made here counts before Zernio lists it.
	feeds := remembered_feed.New(zernioService)

	numbers, err := statsOf(ctx, config, props, clock)
	if err != nil {
		return nil, err
	}

	props.Logger.Info("publishing built", slog.Bool("snapshots", config.Database.Host != ""))
	return publishingv1controller.AdminService{
		CreateMediaUploadHandler: create_media_upload_handler.New(create_upload_usecase.New(zernioService)),
		GetRoomHandler:           get_room_handler.New(room_query.New(feeds, calendar.Default, clock)),
		PlanPostHandler:          plan_post_handler.New(plan_query.New(feeds, calendar.Default, clock)),
		SchedulePostHandler: schedule_post_handler.New(log_schedule_post.New(
			schedule_post_usecase.New(feeds, zernioService, calendar.Default, clock), props.Logger,
		)),
		ListPostsHandler: list_posts_handler.New(zernioService),
		StatsReader:      numbers,
	}, nil
}

// The posts' numbers: a snapshot a day, taken here and kept in publishing's schema, and the reads of them. Off with no
// database: posting still works.
func statsOf(ctx context.Context, config Config, props cpbootstrap.Props, clock cptime.Clock) (publishingv1controller.StatsReader, error) {
	if config.Database.Host == "" {
		props.Logger.Warn("the posts' numbers are off: publishing has no database")
		return publishingv1controller.StatsOff{}, nil
	}
	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect the publishing module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}
	source, err := zernio_source.New(zernio_source.Config{APIKey: config.Zernio.APIKey, BaseURL: config.Zernio.BaseURL},
		&http.Client{Timeout: serviceTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to build the source of the posts' numbers: %w", err)
	}
	store := postgres_snapshot_store.New(db)
	snapshot := log_take_snapshot.New(take_snapshot_usecase.New(
		source, store, clock, time.Time{}, snapshotBackoff, stats.SystemPause{},
	), props.Logger)
	// Not a closer: closers run before the runners stop, and the snapshot may be writing.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, take_snapshot_usecase.NewRunner(snapshot, store, clock, props.Logger)))

	return publishingv1controller.Stats{
		ListSnapshotsHandler: list_snapshots_handler.New(snapshots_query.NewPostgresQuery(db)),
		ListRowsHandler:      list_rows_handler.New(rows_query.NewPostgresQuery(db)),
		GetPostLinksHandler:  get_post_links_handler.New(links_query.NewPostgresQuery(db)),
	}, nil
}

type Config struct {
	// The posting service's. No key, no posting: every call is FailedPrecondition.
	Zernio ZernioConfig
	// Where the daily snapshots of the posts' numbers are kept. No host, no snapshot: posting still works.
	Database cppg.Config
}

type ZernioConfig struct {
	APIKey string
	// Empty is Zernio's own API: a test's fake.
	BaseURL string
}

func (c Config) Validate() error {
	var errs []error
	if c.Zernio.BaseURL != "" {
		if base, err := url.Parse(c.Zernio.BaseURL); err != nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" {
			errs = append(errs, errors.New("publishing.zernio.baseURL is no http address"))
		}
	}
	if c.Database.Host != "" {
		if err := c.Database.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("publishing.database: %w", err))
		}
	}
	return errors.Join(errs...)
}
