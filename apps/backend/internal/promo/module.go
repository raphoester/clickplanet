package promo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1/promov1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/postgres_background_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/s3_background_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/usecases/keep_background_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/usecases/remove_background_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards/postgres_card_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards/rpc_rule_sets"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards/usecases/save_cards_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/placeholders/rpc_scoring_points"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/create_background_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_cards_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_cards_handler/cards_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler/teaser_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler/teaser_query/rpc_gamemaster_battles"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/keep_background_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_backgrounds_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_backgrounds_handler/backgrounds_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_due_teasers_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_due_teasers_handler/due_teasers_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_teasers_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_teasers_handler/teasers_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_teasers_handler/teasers_query/rpc_gamemaster_timeline"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/post_teaser_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/remove_background_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/save_cards_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/listed_battles"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/rpc_publishing"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teaser_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teaser_usecase/log_post_teaser"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "promo"

// What the game shows of a battle before it starts: the teaser's cards of each rule set, over backgrounds the operator
// uploads, with words the set's values fill when a player reads them.
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
	config = config.withDefaults()

	files, err := cps3.New(config.Backgrounds.bucket(), clock, &http.Client{Timeout: bucketTimeout})
	if err != nil {
		return fmt.Errorf("failed to build the bucket of the backgrounds: %w", err)
	}
	bucket := s3_background_bucket.New(files, config.Backgrounds.UploadTTL, config.Backgrounds.PublicBaseURL)
	rules := backgrounds.Rules{MaxBytes: config.Backgrounds.MaxBytes}

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the promo module asks planet what each set plays, the gamemaster what a battle plays, "+
			"scoring what its ranks score and publishing for the teasers' posts: %w", err)
	}
	planet := rpc_rule_sets.New(planetv1connect.NewInternalServiceClient(internal, baseURL),
		bonusesv1connect.NewInternalServiceClient(internal, baseURL))
	gamemaster := gamemasterv1connect.NewInternalServiceClient(internal, baseURL)
	battles := rpc_gamemaster_battles.New(gamemaster)
	points := rpc_scoring_points.New(scoringv1connect.NewInternalServiceClient(internal, baseURL))
	publishing := rpc_publishing.New(publishingv1connect.NewInternalServiceClient(internal, baseURL))

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the promo module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}
	// A closer and not a runner: the module runs nothing, and closers run once the listeners shut down.
	props.Closers.Add("promo postgres", db.Close)

	library := postgres_background_store.New(db)
	teaser := teaser_query.NewPostgresQuery(db, battles, points, planet, bucket, props.Logger)
	listed := teasers_query.New(rpc_gamemaster_timeline.New(gamemaster), teaser, clock)
	admin := promov1controller.AdminService{
		CreateBackgroundUploadHandler: create_background_upload_handler.New(
			create_upload_usecase.New(uuid_id_provider.Provider{}, rules, bucket)),
		KeepBackgroundHandler:   keep_background_handler.New(keep_background_usecase.New(bucket, library, rules, clock)),
		ListBackgroundsHandler:  list_backgrounds_handler.New(backgrounds_query.NewPostgresQuery(db, bucket)),
		RemoveBackgroundHandler: remove_background_handler.New(remove_background_usecase.New(bucket, library)),
		GetCardsHandler:         get_cards_handler.New(cards_query.NewPostgresQuery(db, planet, points, bucket)),
		SaveCardsHandler:        save_cards_handler.New(save_cards_usecase.New(planet, postgres_card_store.New(db), db)),
		ListTeasersHandler:      list_teasers_handler.New(listed),
		ListDueTeasersHandler:   list_due_teasers_handler.New(due_teasers_query.New(listed, publishing, clock)),
		PostTeaserHandler: post_teaser_handler.New(log_post_teaser.New(post_teaser_usecase.New(
			listed_battles.New(listed), publishing, teasers.NewPoster(publishing), clock,
		), props.Logger)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return promov1connect.NewAdminServiceHandler(admin, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount promo.v1.AdminService: %w", err)
	}

	public := promov1controller.PromoService{
		GetTeaserHandler: get_teaser_handler.New(teaser),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return promov1connect.NewPromoServiceHandler(public, options...)
	}, promov1controller.NewCacheInterceptor()); err != nil {
		return fmt.Errorf("failed to mount promo.v1.PromoService: %w", err)
	}

	props.Logger.Info("promo built", slog.String("schema", config.Database.Schema),
		slog.Bool("backgrounds", config.Backgrounds.Enabled))
	return nil
}

type Config struct {
	Database    cppg.Config
	Backgrounds BackgroundsConfig
}

// The teasers' backgrounds, in the support module's bucket under promo/backgrounds/, which Cloudflare serves alone on
// the public domain.
type BackgroundsConfig struct {
	Enabled  bool
	MaxBytes int64
	// The backoffice asks for a link right before it sends the file.
	UploadTTL time.Duration
	// The public domain's origin, such as https://media.clickplanet.lol: a card's background is read from it.
	PublicBaseURL string
	Bucket        cps3.Config
}

// Off, the bucket block is not read, and every background call is FailedPrecondition: no card can be saved, so no
// battle is teased.
func (c BackgroundsConfig) bucket() cps3.Config {
	if !c.Enabled {
		return cps3.Config{}
	}
	return c.Bucket
}

const (
	bucketTimeout = 10 * time.Second

	defaultMaxBackgroundBytes = 2 << 20
	defaultUploadTTL          = 10 * time.Minute

	// A bucket refuses a link signed for longer.
	maxLinkTTL = 7 * 24 * time.Hour
)

func (c Config) withDefaults() Config {
	if c.Backgrounds.MaxBytes <= 0 {
		c.Backgrounds.MaxBytes = defaultMaxBackgroundBytes
	}
	if c.Backgrounds.UploadTTL <= 0 {
		c.Backgrounds.UploadTTL = defaultUploadTTL
	}
	return c
}

func (c Config) Validate() error {
	var errs []error
	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("promo.database: %w", err))
	}
	if !c.Backgrounds.Enabled {
		return errors.Join(errs...)
	}
	if err := c.Backgrounds.Bucket.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("promo.backgrounds.bucket: %w", err))
	}
	if base, err := url.Parse(c.Backgrounds.PublicBaseURL); err != nil || base.Scheme != "https" || base.Host == "" ||
		(base.Path != "" && base.Path != "/") || base.RawQuery != "" {
		errs = append(errs, fmt.Errorf("promo.backgrounds.publicBaseURL is %q, and must be an https origin such as "+
			"https://media.clickplanet.lol", c.Backgrounds.PublicBaseURL))
	}
	if c.Backgrounds.UploadTTL > maxLinkTTL {
		errs = append(errs, fmt.Errorf("promo.backgrounds.uploadTTL is %s, and a bucket takes %s at most",
			c.Backgrounds.UploadTTL, maxLinkTTL))
	}
	return errors.Join(errs...)
}
