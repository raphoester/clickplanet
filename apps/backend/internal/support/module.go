package support

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/postgres_note_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/s3_attachment_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/anonymize_notes_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/prune_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/prune_usecase/log_prune"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/report_bug_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/suggest_feature_usecase"
	note_ids "github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/postgres_report_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/usecases/forget_reports_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/usecases/prune_reports_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/usecases/report_player_usecase"
	report_ids "github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/create_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/get_attachment_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_notes_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_notes_handler/notes_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_reports_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_reports_handler/reports_query"
	report_authors "github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_reports_handler/reports_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/report_bug_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/report_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/suggest_feature_handler"
)

const moduleName = "support"

const accountDeletedBuffer = 256

const bucketTimeout = 4 * time.Second

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

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the support module asks the player module how to name the accounts of a report: %w", err)
	}
	player := playerv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the support module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	noteStore := postgres_note_store.New(db)
	reportStore := postgres_report_store.New(db)

	bucket, err := s3_attachment_bucket.New(config.Attachments.bucket(), config.Attachments.lifetimes(), clock,
		&http.Client{Timeout: bucketTimeout})
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to build the bucket of the attachments: %w", err)
	}
	rules := config.Attachments.rules()

	notePrune := log_prune.New(prune_usecase.New(config.Retention, clock, noteStore),
		props.Logger.With(slog.String("pruned", "notes")))
	reportPrune := log_prune.New(prune_reports_usecase.New(config.Retention, clock, reportStore),
		props.Logger.With(slog.String("pruned", "reports")))

	noteDeletions, err := cpbootstrap.Subscribe(props.Events, "support-notes-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(anonymize_notes_usecase.New(noteStore)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}

	reportDeletions, err := cpbootstrap.Subscribe(props.Events, "support-reports-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_reports_usecase.New(reportStore)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}

	// Not a closer: closers run before the runners stop, and the runners use the pool.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger,
		prune_usecase.NewRunner(config.PruneInterval, notePrune, reportPrune), noteDeletions, reportDeletions))

	limiter := cpratelimit.New("support-limiter", config.RateLimiter, clock)
	props.Runners.Add(limiter)
	uploadLimiter := cpratelimit.New("support-upload-limiter", config.UploadLimiter, clock)
	props.Runners.Add(uploadLimiter)

	service := supportv1controller.SupportService{
		SuggestFeatureHandler: suggest_feature_handler.New(
			suggest_feature_usecase.New(note_ids.Provider{}, clock, rules, noteStore, bucket)),
		ReportBugHandler: report_bug_handler.New(
			report_bug_usecase.New(note_ids.Provider{}, clock, rules, noteStore, bucket)),
		ReportPlayerHandler: report_player_handler.New(report_player_usecase.New(
			report_ids.Provider{}, clock, reportStore)),
		GetAttachmentRulesHandler: get_attachment_rules_handler.New(rules),
		CreateUploadHandler:       create_upload_handler.New(create_upload_usecase.New(note_ids.Provider{}, rules, bucket)),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", moduleName)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return supportv1connect.NewSupportServiceHandler(service, options...)
	},
		supportv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
		supportv1controller.NewRateLimitInterceptor(limiter),
		supportv1controller.NewUploadRateLimitInterceptor(uploadLimiter),
	); err != nil {
		return fmt.Errorf("failed to mount support.v1.SupportService: %w", err)
	}

	adminService := supportv1controller.AdminService{
		ListNotesHandler:   list_notes_handler.New(notes_query.NewPostgresQuery(db, bucket)),
		ListReportsHandler: list_reports_handler.New(reports_query.NewPostgresQuery(db, report_authors.New(player))),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return supportv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount support.v1.AdminService: %w", err)
	}

	props.Logger.Info("support built",
		slog.String("schema", config.Database.Schema), slog.Bool("attachments", config.Attachments.Enabled))

	return nil
}

type Config struct {
	Database cppg.Config

	Retention     time.Duration
	PruneInterval time.Duration

	RateLimiter   cpratelimit.Config
	UploadLimiter cpratelimit.Config

	Attachments AttachmentsConfig
}

type AttachmentsConfig struct {
	Enabled bool

	MaxPerNote int
	MaxBytes   int64

	UploadTTL time.Duration
	LinkTTL   time.Duration

	Bucket s3_attachment_bucket.Config
}

// Off, a note carries no file, and the bucket block is not read.
func (c AttachmentsConfig) rules() notes.AttachmentRules {
	if !c.Enabled {
		return notes.AttachmentRules{}
	}
	return notes.AttachmentRules{MaxPerNote: c.MaxPerNote, MaxBytes: c.MaxBytes}
}

func (c AttachmentsConfig) bucket() s3_attachment_bucket.Config {
	if !c.Enabled {
		return s3_attachment_bucket.Config{}
	}
	return c.Bucket
}

func (c AttachmentsConfig) lifetimes() s3_attachment_bucket.Lifetimes {
	return s3_attachment_bucket.Lifetimes{Upload: c.UploadTTL, View: c.LinkTTL}
}

const (
	defaultRetention     = 90 * 24 * time.Hour
	defaultPruneInterval = time.Hour
	defaultBurst         = 3

	defaultMaxPerNote = 10
	defaultMaxBytes   = 50 << 20
	defaultUploadTTL  = 30 * time.Minute
	defaultLinkTTL    = time.Hour

	// A bucket refuses a link signed for longer.
	maxLinkTTL = 7 * 24 * time.Hour
)

const defaultPerSecond = 1.0 / 60

const (
	defaultUploadBurst      = 10
	defaultUploadsPerSecond = 1.0 / 6
)

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
	if c.UploadLimiter.PerSecond <= 0 {
		c.UploadLimiter.PerSecond = defaultUploadsPerSecond
	}
	if c.UploadLimiter.Burst <= 0 {
		c.UploadLimiter.Burst = defaultUploadBurst
	}
	if c.Attachments.MaxPerNote <= 0 {
		c.Attachments.MaxPerNote = defaultMaxPerNote
	}
	if c.Attachments.MaxBytes <= 0 {
		c.Attachments.MaxBytes = defaultMaxBytes
	}
	if c.Attachments.UploadTTL <= 0 {
		c.Attachments.UploadTTL = defaultUploadTTL
	}
	if c.Attachments.LinkTTL <= 0 {
		c.Attachments.LinkTTL = defaultLinkTTL
	}
	return c
}

func (c Config) Validate() error {
	var errs []error
	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("support.database: %w", err))
	}
	if !c.Attachments.Enabled {
		return errors.Join(errs...)
	}

	if err := c.Attachments.Bucket.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("support.attachments.bucket: %w", err))
	}
	for name, ttl := range map[string]time.Duration{"uploadTTL": c.Attachments.UploadTTL, "linkTTL": c.Attachments.LinkTTL} {
		if ttl > maxLinkTTL {
			errs = append(errs, fmt.Errorf("support.attachments.%s is %s, and a bucket takes %s at most", name, ttl, maxLinkTTL))
		}
	}
	return errors.Join(errs...)
}
