package support

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/postgres_note_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/anonymize_notes_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/prune_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/prune_usecase/log_prune"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/report_bug_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/suggest_feature_usecase"
	note_ids "github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_notes_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_notes_handler/notes_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/report_bug_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/suggest_feature_handler"
)

const moduleName = "support"

const accountDeletedBuffer = 256

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
	clock := cptime.SystemClock{}

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the support module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	noteStore := postgres_note_store.New(db)

	prune := log_prune.New(prune_usecase.New(config.Retention, clock, noteStore), props.Logger)

	noteDeletions, err := cpbootstrap.Subscribe(props.Events, "support-notes-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(anonymize_notes_usecase.New(noteStore)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}

	// Not a closer: closers run before the runners stop, and the runners use the pool.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger,
		prune_usecase.NewRunner(config.PruneInterval, prune), noteDeletions))

	limiter := cpratelimit.New("support-limiter", config.RateLimiter, clock)
	props.Runners.Add(limiter)

	service := supportv1controller.SupportService{
		SuggestFeatureHandler: suggest_feature_handler.New(suggest_feature_usecase.New(note_ids.Provider{}, clock, noteStore)),
		ReportBugHandler:      report_bug_handler.New(report_bug_usecase.New(note_ids.Provider{}, clock, noteStore)),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", moduleName)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return supportv1connect.NewSupportServiceHandler(service, options...)
	},
		supportv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
		supportv1controller.NewRateLimitInterceptor(limiter),
	); err != nil {
		return fmt.Errorf("failed to mount support.v1.SupportService: %w", err)
	}

	adminService := supportv1controller.AdminService{
		ListNotesHandler: list_notes_handler.New(notes_query.NewPostgresQuery(db)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return supportv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount support.v1.AdminService: %w", err)
	}

	props.Logger.Info("support built", slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Database cppg.Config

	Retention     time.Duration
	PruneInterval time.Duration

	RateLimiter cpratelimit.Config
}

const (
	defaultRetention     = 90 * 24 * time.Hour
	defaultPruneInterval = time.Hour
	defaultBurst         = 3
)

const defaultPerSecond = 1.0 / 60

func (c Config) withDefaults() Config {
	if c.Retention <= 0 {
		c.Retention = defaultRetention
	}
	if c.PruneInterval <= 0 {
		c.PruneInterval = defaultPruneInterval
	}
	if c.RateLimiter.PerSecond <= 0 {
		c.RateLimiter.PerSecond = defaultPerSecond
	}
	if c.RateLimiter.Burst <= 0 {
		c.RateLimiter.Burst = defaultBurst
	}
	return c
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("support.database: %w", err)
	}
	return nil
}
