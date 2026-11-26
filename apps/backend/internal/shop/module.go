package shop

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/stripe/stripe-go/v87"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1/flagsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/shop/v1/shopv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/checkouts/postgres_checkout_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices/postgres_inbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices/usecases/receive_notice_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices/usecases/receive_notice_usecase/log_receive_notice"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders/usecases/create_checkout_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/postgres_payment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/pay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/refund_refused_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/refund_refused_usecase/log_refund_refused"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/refund_rejected_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/refund_rejected_usecase/log_refund_rejected"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/take_back_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals/postgres_renewal_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals/usecases/hear_renewal_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals/usecases/stop_renewal_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/rpc_payables"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/create_checkout_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/list_my_renewals_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/list_my_renewals_handler/renewals_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/list_plans_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/stop_renewal_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/stripe_gateway"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/stripecontroller/receive_notice_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/subscribers/refused_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/subscribers/rejected_subscriber"
)

const moduleName = "shop"

const (
	stripeTimeout = 15 * time.Second
	// A Stripe event is a few kilobytes; an invoice with its lines, a few tens.
	maxNoticeBytes = 1 << 20
)

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config, props)
		},
	}
}

// Stripe's API, in the shop's words.
type gateway interface {
	create_checkout_usecase.Pages
	stop_renewal_usecase.Subscriptions
	pay_usecase.Intents
	refund_rejected_usecase.Refunder
}

func build(ctx context.Context, config Config, props cpbootstrap.Props) error {
	clock := cptime.SystemClock{}
	catalog, err := config.Stripe.catalog()
	if err != nil {
		return fmt.Errorf("shop.stripe: %w", err)
	}
	var stripeAPI gateway = stripe_gateway.Off{}
	if config.Stripe.on() {
		returnURL, err := url.Parse(config.Stripe.ReturnURL)
		if err != nil {
			return fmt.Errorf("shop.stripe.returnURL: %w", err)
		}
		stripeAPI = stripe_gateway.New(stripe.NewClient(config.Stripe.SecretKey, stripe.WithBackends(
			stripe.NewBackendsWithConfig(&stripe.BackendConfig{HTTPClient: &http.Client{Timeout: stripeTimeout}}),
		)), returnURL)
	}

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the shop asks the flags module: %w", err)
	}
	flags := flagsv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the shop to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}
	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the shop to redis: %w", err)
	}
	paid, renewed, opened := postgres_payment_store.New(db), postgres_renewal_store.New(db), postgres_checkout_store.New(db)
	// A flag rejected after it was paid for is refunded, and its renewal cancelled.
	rejected, err := cpredis.Subscribe(ctx, streams, "shop-refund-rejected", rejected_subscriber.New(
		log_refund_rejected.New(refund_rejected_usecase.New(clock, opened, paid, renewed, stripeAPI), props.Logger),
	))
	if err != nil {
		_ = streams.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to flags.v1.SubmissionRejected: %w", err)
	}
	// A payment the flags module refused bought nothing: it is refunded, and its renewal cancelled.
	refused, err := cpredis.Subscribe(ctx, streams, "shop-refund-refused", refused_subscriber.New(
		log_refund_refused.New(refund_refused_usecase.New(paid, renewed, stripeAPI), props.Logger)))
	if err != nil {
		_ = streams.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to flags.v1.TimeRefused: %w", err)
	}
	// Not a closer: closers run before the runners stop, and the relay uses the pool.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger, rejected, refused,
		cpoutbox.NewRelay("shop-outbox", db, streams.Commands(), props.Logger))))

	if err := props.RPC.Mount(
		func(options ...connect.HandlerOption) (string, http.Handler) {
			return shopv1connect.NewShopServiceHandler(shopv1controller.ShopService{
				ListPlansHandler: list_plans_handler.New(catalog, clock),
				CreateCheckoutHandler: create_checkout_handler.New(create_checkout_usecase.New(
					clock, catalog, rpc_payables.New(flags),
					create_checkout_usecase.Stores{Renewals: renewed, Checkouts: opened}, stripeAPI,
				)),
				ListMyRenewalsHandler: list_my_renewals_handler.New(renewals_query.NewPostgresQuery(db)),
				StopRenewalHandler:    stop_renewal_handler.New(stop_renewal_usecase.New(clock, renewed, stripeAPI)),
			}, options...)
		},
		shopv1controller.NewSessionInterceptor(props.Sessions, clock, props.Metrics),
	); err != nil {
		return fmt.Errorf("failed to mount shop.v1.ShopService: %w", err)
	}

	// No secret, no notice: a request nobody can sign is never served.
	if config.Stripe.on() {
		inbox := postgres_inbox.New(db)
		receive := log_receive_notice.New(receive_notice_usecase.New(
			pay_usecase.New(db, inbox, paid, stripeAPI, cpoutbox.New(db)),
			take_back_usecase.New(db, inbox, paid, cpoutbox.New(db)),
			hear_renewal_usecase.New(db, inbox, renewed),
		), props.Logger)
		if err := props.AdminHTTP.Handle("POST "+receive_notice_handler.Path, receive_notice_handler.New(
			stripe_gateway.NewReader(config.Stripe.WebhookSecret), receive, maxNoticeBytes,
		)); err != nil {
			return fmt.Errorf("failed to mount Stripe's webhook endpoint: %w", err)
		}
	}

	props.Logger.Info("shop built", slog.String("schema", config.Database.Schema), slog.Bool("on", config.Stripe.on()))
	return nil
}

type Config struct {
	// What the shop sold and the outbox of what it asks the flags module.
	Database cppg.Config
	// Where it asks the flags module for the time each payment bought.
	Redis  cpredis.Config
	Stripe StripeConfig
}

// The shop is off with no key or no webhook secret: no plan is on sale and no Stripe event is read.
type StripeConfig struct {
	// A restricted key (rk_) or a secret key (sk_), with the rights the shop uses: Checkout Sessions, Subscriptions and
	// Refunds written, Invoices read.
	SecretKey string
	// The webhook endpoint's signing secret (whsec_).
	WebhookSecret string
	// The game's page Stripe sends the buyer back to, with ?shop=paid or ?shop=cancelled.
	ReturnURL string
	Plans     PlansConfig
	// None: no sale.
	Sale SaleConfig
}

// A Stripe coupon taken off every plan's first payment, from one moment until another.
type SaleConfig struct {
	// Made by the owner in Stripe, limited to the flags' products, redeemable until Until.
	Coupon string
	// What the coupon takes off, as the page shows it: 50 is half.
	PercentOff uint32
	From       time.Time
	// The end, excluded: the coupon's redeem_by.
	Until time.Time
}

// Each plan's Stripe price, and what it charges, taxes included, as the page shows it.
type PlansConfig struct {
	Week  PriceConfig
	Month PriceConfig
	Year  PriceConfig
}

type PriceConfig struct {
	PriceID string
	// In the currency's smallest unit: 400 is 4.00 EUR.
	Amount   uint32
	Currency string
}

// Sells is true while the shop is on: a flag is then paid for before its review.
func (c Config) Sells() bool {
	return c.Stripe.on()
}

func (c StripeConfig) on() bool {
	return c.SecretKey != "" && c.WebhookSecret != ""
}

func (c StripeConfig) catalog() (plans.Catalog, error) {
	if !c.on() {
		return plans.Catalog{}, nil
	}
	sale, err := c.Sale.sale()
	if err != nil {
		return plans.Catalog{}, fmt.Errorf("sale: %w", err)
	}
	catalog, err := plans.NewCatalog(
		sale,
		plans.Price{
			Plan: plans.Week, ID: plans.PriceID(c.Plans.Week.PriceID), Amount: c.Plans.Week.Amount,
			Currency: c.Plans.Week.Currency,
		},
		plans.Price{
			Plan: plans.Month, ID: plans.PriceID(c.Plans.Month.PriceID), Amount: c.Plans.Month.Amount,
			Currency: c.Plans.Month.Currency,
		},
		plans.Price{
			Plan: plans.Year, ID: plans.PriceID(c.Plans.Year.PriceID), Amount: c.Plans.Year.Amount,
			Currency: c.Plans.Year.Currency,
		},
	)
	if err != nil {
		return plans.Catalog{}, fmt.Errorf("plans: %w", err)
	}
	return catalog, nil
}

func (c SaleConfig) sale() (plans.Sale, error) {
	if c == (SaleConfig{}) {
		return plans.Sale{}, nil
	}
	return plans.SaleSnapshot{ //nolint:wrapcheck // the sale names what is wrong.
		Coupon: plans.CouponID(c.Coupon), PercentOff: c.PercentOff, From: c.From, Until: c.Until,
	}.Restore()
}

func (c Config) Validate() error {
	var errs []error
	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("shop.database: %w", err))
	}
	if err := c.Redis.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("shop.redis: %w", err))
	}
	stripeConfig := c.Stripe
	if (stripeConfig.SecretKey == "") != (stripeConfig.WebhookSecret == "") {
		errs = append(errs, errors.New("shop.stripe: give both secretKey and webhookSecret, or neither to keep the "+
			"shop off"))
	}
	if !stripeConfig.on() {
		return errors.Join(errs...)
	}
	if !strings.HasPrefix(stripeConfig.SecretKey, "sk_") && !strings.HasPrefix(stripeConfig.SecretKey, "rk_") {
		errs = append(errs, errors.New("shop.stripe.secretKey is no Stripe key: sk_ or rk_"))
	}
	if !strings.HasPrefix(stripeConfig.WebhookSecret, "whsec_") {
		errs = append(errs, errors.New("shop.stripe.webhookSecret is no webhook secret: whsec_"))
	}
	if back, err := url.Parse(stripeConfig.ReturnURL); err != nil || back.Host == "" ||
		(back.Scheme != "https" && back.Hostname() != "localhost") {
		errs = append(errs, fmt.Errorf("shop.stripe.returnURL is %q, and must be the game's page, such as "+
			"https://clickplanet.lol/play", stripeConfig.ReturnURL))
	}
	if _, err := stripeConfig.catalog(); err != nil {
		errs = append(errs, fmt.Errorf("shop.stripe: %w", err))
	}
	return errors.Join(errs...)
}
