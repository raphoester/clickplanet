package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/random_token_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/create_anonymous_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/create_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/delete_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/get_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/get_accounts_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/get_me_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/prune_guests_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/prune_guests_usecase/log_prune_guests"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/sign_out_everywhere_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/sign_out_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/open_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/turnstile"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/turnstile_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_email_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/create_session_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/delete_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_accounts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_sign_in_options_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_verifying_key_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_everywhere_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_email_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/sessionv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/cloudflare_mailer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/discord_identity_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/embedded_disposable_domains"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/google_identity_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/log_mailer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/random_code_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/random_secret_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_email_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_email_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "auth"

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			config := config.withDefaults()
			mailer, err := newMailer(config.Email, props.Logger)
			if err != nil {
				return err
			}
			return build(ctx, config, props, newProviders(config), mailing{mailer: mailer, codes: random_code_generator.Generator{}})
		},
	}
}

const providerTimeout = 10 * time.Second

func newProviders(config Config) signin.Providers {
	providers := signin.Providers{}
	if !config.SignIn.Enabled {
		return providers
	}

	httpClient := &http.Client{Timeout: providerTimeout}
	if config.Google.Configured() {
		providers[signin.Google] = google_identity_provider.New(config.Google, config.SignIn.RedirectURL,
			google_identity_provider.Production, httpClient, cptime.SystemClock{})
	}
	if config.Discord.Configured() {
		providers[signin.Discord] = discord_identity_provider.New(config.Discord, config.SignIn.RedirectURL,
			discord_identity_provider.Production, httpClient)
	}
	return providers
}

const mailerTimeout = 10 * time.Second

const (
	deliveryCloudflare = "cloudflare"
	deliveryLog        = "log"
)

type mailing struct {
	mailer signin.Mailer
	codes  signin.Codes
}

func newMailer(config EmailConfig, logger *slog.Logger) (signin.Mailer, error) {
	if !config.Enabled || config.Delivery != deliveryCloudflare {
		if config.Enabled {
			logger.Warn("auth.email.delivery is log: sign-in codes are written to the log, not sent")
		}
		return log_mailer.New(logger), nil
	}

	mailer, err := cloudflare_mailer.New(config.Cloudflare, cloudflare_mailer.Sender{Address: config.From, Name: config.FromName},
		cloudflare_mailer.Production, &http.Client{Timeout: mailerTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to build the cloudflare mailer: %w", err)
	}
	return mailer, nil
}

func build(ctx context.Context, config Config, props cpbootstrap.Props, providers signin.Providers, mail mailing) error {
	signer, err := cpsession.NewSigner(config.SignerConfig)
	if err != nil {
		return fmt.Errorf("failed to build the click token signer: %w", err)
	}

	attester, err := newAttester(config, props.Logger)
	if err != nil {
		return err
	}

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect auth to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}
	props.Closers.Add("auth-postgres", db.Close)

	store := postgres_account_store.New(db)
	clock := cptime.SystemClock{}

	// One budget for both mints, so the deprecated path is not a second allowance.
	mintLimiter := cpratelimit.New("mint-limiter", config.RateLimiter, clock)
	props.Runners.Add(mintLimiter)

	seed, err := hex.DecodeString(config.Secret)
	if err != nil {
		return fmt.Errorf("failed to read auth.secret: %w", err)
	}
	sealer, err := aes_flow_sealer.New(seed)
	if err != nil {
		return fmt.Errorf("failed to build the sign-in cookie sealer: %w", err)
	}

	blocklist := embedded_disposable_domains.New()
	if err := blocklist.Load(); err != nil {
		return fmt.Errorf("failed to load the disposable domains: %w", err)
	}
	sendLimiter := cpratelimit.New("email-send-limiter", config.Email.SendLimiter, clock)
	props.Runners.Add(sendLimiter)
	attemptLimiter := cpratelimit.New("email-attempt-limiter",
		cpratelimit.Config{Burst: signin.MaxAttempts, PerSecond: 1 / signin.ChallengeTTL.Seconds()}, clock)
	props.Runners.Add(attemptLimiter)
	admitter := signin.NewAdmitter(store, uuid_id_provider.Provider{}, random_token_generator.Generator{}, config.Sessions, props.Events)
	challenges := signin.NewChallenges(config.Email.Enabled, random_secret_generator.Generator{}, mail.codes, sealer, attemptLimiter)
	post := signin.NewPost(blocklist, sendLimiter, mail.mailer)

	authService := authv1controller.AuthService{
		CreateSessionHandler: create_session_handler.New(
			create_session_usecase.New(attester, store, uuid_id_provider.Provider{}, random_token_generator.Generator{},
				signer, config.Sessions, clock),
			props.Logger,
		),
		GetMeHandler:            get_me_handler.New(get_me_usecase.New(store, clock)),
		GetSignInOptionsHandler: get_sign_in_options_handler.New(signin.Offer{Providers: providers, Email: config.Email.Enabled}),
		StartSignInHandler: start_sign_in_handler.New(
			start_sign_in_usecase.New(providers, store, random_secret_generator.Generator{}, sealer, clock),
		),
		CompleteSignInHandler: complete_sign_in_handler.New(
			complete_sign_in_usecase.New(providers, sealer, admitter, clock),
			props.Logger,
		),
		StartEmailSignInHandler: start_email_sign_in_handler.New(
			start_email_sign_in_usecase.New(attester, store, challenges, post, clock),
			props.Logger,
		),
		CompleteEmailSignInHandler: complete_email_sign_in_handler.New(
			complete_email_sign_in_usecase.New(challenges, admitter, clock),
			props.Logger,
		),
		SignOutHandler:           sign_out_handler.New(sign_out_usecase.New(store, props.Events)),
		SignOutEverywhereHandler: sign_out_everywhere_handler.New(sign_out_everywhere_usecase.New(store, props.Events, clock)),
		DeleteAccountHandler:     delete_account_handler.New(delete_account_usecase.New(store, props.Events, clock)),
	}
	props.Runners.Add(prune_guests_usecase.NewRunner(config.Prune,
		log_prune_guests.New(prune_guests_usecase.New(config.Prune, store, props.Events, clock), props.Logger)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return authv1connect.NewAuthServiceHandler(authService, options...)
	}, authv1controller.NewRateLimitInterceptor(mintLimiter)); err != nil {
		return fmt.Errorf("failed to mount auth.v1: %w", err)
	}

	internalService := authv1controller.InternalService{
		GetVerifyingKeyHandler: get_verifying_key_handler.New(signer),
		GetAccountHandler:      get_account_handler.New(get_account_usecase.New(store)),
		GetAccountsHandler:     get_accounts_handler.New(get_accounts_usecase.New(store)),
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return authv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount auth.v1.InternalService: %w", err)
	}

	sessionService := sessionv1controller.NewSessionService(
		create_anonymous_session_usecase.New(attester, signer, clock), props.Logger,
	)
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return sessionv1connect.NewSessionServiceHandler(sessionService, options...)
	}, sessionv1controller.NewRateLimitInterceptor(mintLimiter)); err != nil {
		return fmt.Errorf("failed to mount the deprecated session.v1: %w", err)
	}

	props.Logger.Info(
		"auth built",
		slog.String("schema", config.Database.Schema),
		slog.Duration("ttl", config.TTL),
		slog.Bool("turnstile", config.Turnstile.Enabled),
		slog.Duration("guestTTL", config.Sessions.GuestTTL),
		slog.Duration("linkedTTL", config.Sessions.LinkedTTL),
		slog.Any("signIn", signin.Offer{Providers: providers, Email: config.Email.Enabled}.Names()),
		slog.String("emailDelivery", config.Email.Delivery),
		slog.Int("disposableDomains", blocklist.Size()),
		slog.Duration("pruneIdleFor", config.Prune.IdleFor),
	)

	return nil
}

func newAttester(config Config, logger *slog.Logger) (attestation.Attester, error) {
	if !config.Turnstile.Enabled {
		logger.Warn("auth.turnstile is disabled: a session is minted for anyone who asks for one")
		return open_attester.New(), nil
	}

	attester, err := turnstile_attester.New(config.Turnstile)
	if err != nil {
		return nil, fmt.Errorf("failed to build the turnstile attester: %w", err)
	}
	return attester, nil
}

type Config struct {
	cpsession.SignerConfig `koanf:",squash"`

	RateLimiter cpratelimit.Config

	Turnstile turnstile.Config

	Database cppg.Config

	Sessions accounts.Lifetime

	SignIn signin.Config

	Google  signin.Client
	Discord signin.Client

	Email EmailConfig

	Prune prune_guests_usecase.Config
}

type EmailConfig struct {
	Enabled bool

	Delivery string

	From     string
	FromName string

	Cloudflare cloudflare_mailer.Config

	SendLimiter cpratelimit.Config
}

const defaultTurnstileAction = "session"

func (c Config) withDefaults() Config {
	if c.Turnstile.Action == "" {
		c.Turnstile.Action = defaultTurnstileAction
	}
	c.Sessions = c.Sessions.WithDefaults()
	if c.Prune.IdleFor <= 0 {
		c.Prune.IdleFor = c.Sessions.GuestTTL
	}
	c.Prune = c.Prune.WithDefaults()
	if c.Email.SendLimiter.Burst <= 0 {
		c.Email.SendLimiter.Burst = 3
	}
	if c.Email.SendLimiter.PerSecond <= 0 {
		c.Email.SendLimiter.PerSecond = 1.0 / (20 * time.Minute).Seconds()
	}
	return c
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	var databaseErr error
	if err := c.Database.Validate(); err != nil {
		databaseErr = fmt.Errorf("auth.database: %w", err)
	}
	return errors.Join(c.SignerConfig.Validate(), databaseErr, c.pruneError(), c.signInError(), c.emailError())
}

func (c Config) pruneError() error {
	config := c.withDefaults()
	if config.Prune.IdleFor < config.Sessions.GuestTTL {
		return fmt.Errorf("auth.prune.idleFor %s is shorter than auth.sessions.guestTTL %s: a live cookie would lose its account",
			config.Prune.IdleFor, config.Sessions.GuestTTL)
	}
	return nil
}

func (c Config) signInError() error {
	if !c.SignIn.Enabled {
		return nil
	}

	var errs []error
	redirect, err := url.Parse(c.SignIn.RedirectURL)
	if err != nil || !redirect.IsAbs() || redirect.Host == "" || redirect.RawQuery != "" || redirect.Fragment != "" {
		errs = append(errs, fmt.Errorf("auth.signIn.redirectUrl %q is not an absolute URL without a query", c.SignIn.RedirectURL))
	}

	offered := 0
	for name, client := range map[string]signin.Client{"google": c.Google, "discord": c.Discord} {
		if !client.Configured() {
			continue
		}
		offered++
		if client.ClientSecret == "" {
			errs = append(errs, fmt.Errorf("auth.%s.clientSecret is empty while auth.%s.clientId is set", name, name))
		}
	}
	if offered == 0 {
		errs = append(errs, errors.New("auth.signIn.enabled is true and no provider has a clientId"))
	}
	return errors.Join(errs...)
}

func (c Config) emailError() error {
	if !c.Email.Enabled {
		return nil
	}

	switch c.Email.Delivery {
	case deliveryLog:
		return nil
	case deliveryCloudflare:
	default:
		return fmt.Errorf("auth.email.delivery %q is neither %q nor %q", c.Email.Delivery, deliveryCloudflare, deliveryLog)
	}

	var errs []error
	if _, err := signin.AddressOf(c.Email.From); err != nil {
		errs = append(errs, fmt.Errorf("auth.email.from %q: %w", c.Email.From, err))
	}
	if c.Email.Cloudflare.AccountID == "" {
		errs = append(errs, errors.New("auth.email.cloudflare.accountId is empty while auth.email.delivery is cloudflare"))
	}
	if c.Email.Cloudflare.APIToken == "" {
		errs = append(errs, errors.New("auth.email.cloudflare.apiToken is empty while auth.email.delivery is cloudflare"))
	}
	return errors.Join(errs...)
}
