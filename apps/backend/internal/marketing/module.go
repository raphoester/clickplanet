package marketing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1/marketingv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/get_subscription_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/rpc_session_verifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/subscribe_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/unsubscribe_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/unsubscribed_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/brevo_audience"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/log_audience"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/log_failing_audience"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/postgres_subscription_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/rpc_account_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/get_subscription_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/subscribe_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/unsubscribe_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/withdraw_address_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "marketing"

const accountDeletedBuffer = 2048

const (
	deliveryBrevo = "brevo"
	deliveryLog   = "log"
)

const brevoTimeout = 5 * time.Second

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			audience, err := newAudience(config.Audience, props.Logger)
			if err != nil {
				return err
			}
			return build(ctx, config.withDefaults(), props, audience)
		},
	}
}

func newAudience(config AudienceConfig, logger *slog.Logger) (subscriptions.Audience, error) {
	if config.Delivery != deliveryBrevo {
		logger.Warn("marketing.audience.delivery is log: season email opt-ins are written to the log, not sent to Brevo")
		return log_audience.New(logger), nil
	}

	audience, err := brevo_audience.New(config.Brevo, brevo_audience.Production, &http.Client{Timeout: brevoTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to build the brevo audience: %w", err)
	}
	return audience, nil
}

func build(ctx context.Context, config Config, props cpbootstrap.Props, audience subscriptions.Audience) error {
	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the marketing module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	store := postgres_subscription_store.New(db)
	clock := cptime.SystemClock{}
	accounts := rpc_account_reader.New(props.Internal)
	audience = log_failing_audience.New(audience, props.Logger)

	deletions, err := cpbootstrap.Subscribe(props.Events, "marketing-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(store, audience)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, deletions))

	limiter := cpratelimit.New("marketing-limiter", config.RateLimiter, clock)
	props.Runners.Add(limiter)

	verifier := rpc_session_verifier.New(props.Internal, props.Logger)

	subscriptionService := marketingv1controller.SubscriptionService{
		GetSubscriptionHandler: get_subscription_handler.New(get_subscription_usecase.New(accounts, store, audience)),
		SubscribeHandler:       subscribe_handler.New(subscribe_usecase.New(accounts, store, audience, clock)),
		UnsubscribeHandler:     unsubscribe_handler.New(unsubscribe_usecase.New(store, audience, clock)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return marketingv1connect.NewSubscriptionServiceHandler(subscriptionService, options...)
	},
		marketingv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
		marketingv1controller.NewRateLimitInterceptor(limiter),
	); err != nil {
		return fmt.Errorf("failed to mount marketing.v1.SubscriptionService: %w", err)
	}

	brevoService := marketingv1controller.BrevoService{
		UnsubscribedHandler: unsubscribed_handler.New(withdraw_address_usecase.New(store, clock)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return marketingv1connect.NewBrevoServiceHandler(brevoService, options...)
	}, marketingv1controller.NewWebhookInterceptor(config.WebhookSecret)); err != nil {
		return fmt.Errorf("failed to mount marketing.v1.BrevoService: %w", err)
	}

	props.Logger.Info("marketing built",
		slog.String("schema", config.Database.Schema),
		slog.String("delivery", config.Audience.Delivery),
	)

	return nil
}

type Config struct {
	Enabled bool

	Database cppg.Config

	Audience AudienceConfig

	WebhookSecret string

	RateLimiter cpratelimit.Config
}

type AudienceConfig struct {
	Delivery string

	Brevo brevo_audience.Config
}

// String leaves out the webhook secret: the config is logged at boot.
func (c Config) String() string {
	return fmt.Sprintf("{Enabled:%t Database:%s Audience:%+v RateLimiter:%+v}", c.Enabled, c.Database, c.Audience, c.RateLimiter)
}

func (c Config) withDefaults() Config {
	if c.RateLimiter.Burst <= 0 {
		c.RateLimiter.Burst = 5
	}
	if c.RateLimiter.PerSecond <= 0 {
		c.RateLimiter.PerSecond = 1.0 / time.Minute.Seconds()
	}
	return c
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	var errs []error
	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("marketing.database: %w", err))
	}
	if c.WebhookSecret == "" {
		errs = append(errs, errors.New("marketing.webhookSecret is empty: anybody could withdraw anybody"))
	}

	switch c.Audience.Delivery {
	case deliveryLog:
	case deliveryBrevo:
		brevo := c.Audience.Brevo
		if brevo.APIKey == "" {
			errs = append(errs, errors.New("marketing.audience.brevo.apiKey is empty while marketing.audience.delivery is brevo"))
		}
		if brevo.ListID <= 0 {
			errs = append(errs, errors.New("marketing.audience.brevo.listId is not set while marketing.audience.delivery is brevo"))
		}
		if brevo.DOITemplateID <= 0 {
			errs = append(errs, errors.New("marketing.audience.brevo.doiTemplateId is not set while marketing.audience.delivery is brevo"))
		}
	default:
		errs = append(errs, fmt.Errorf("marketing.audience.delivery %q is neither %q nor %q", c.Audience.Delivery, deliveryBrevo, deliveryLog))
	}
	return errors.Join(errs...)
}
