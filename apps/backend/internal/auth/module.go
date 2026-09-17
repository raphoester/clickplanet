// Package auth wires who a caller is and what it has to prove before it may
// click: Turnstile, the account its cookie holds, and the click token it mints.
//
// The planet context verifies that token from the same `auth:` block, and
// builds its own signer from it rather than being handed this one.
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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/create_session_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/delete_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_sign_in_options_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_verifying_key_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_everywhere_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/sessionv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/discord_identity_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/google_identity_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/random_secret_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_sign_in_usecase"
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
			return build(ctx, config, props, newProviders(config))
		},
	}
}

// providerTimeout bounds each call to a provider, so a slow one cannot hold a sign-in open.
const providerTimeout = 10 * time.Second

// newProviders is every provider with a client id, or none while sign-in is off.
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

func build(ctx context.Context, config Config, props cpbootstrap.Props, providers signin.Providers) error {
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

	// One budget for both mints: the deprecated path must not be a second allowance.
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

	authService := authv1controller.AuthService{
		CreateSessionHandler: create_session_handler.New(
			create_session_usecase.New(attester, store, uuid_id_provider.Provider{}, random_token_generator.Generator{},
				signer, config.Sessions, clock),
			props.Logger,
		),
		GetMeHandler:            get_me_handler.New(get_me_usecase.New(store, clock)),
		GetSignInOptionsHandler: get_sign_in_options_handler.New(providers),
		StartSignInHandler: start_sign_in_handler.New(
			start_sign_in_usecase.New(providers, store, random_secret_generator.Generator{}, sealer, clock),
		),
		CompleteSignInHandler: complete_sign_in_handler.New(
			complete_sign_in_usecase.New(providers, sealer, store, uuid_id_provider.Provider{}, random_token_generator.Generator{},
				config.Sessions, clock),
			props.Logger,
		),
		SignOutHandler:           sign_out_handler.New(sign_out_usecase.New(store)),
		SignOutEverywhereHandler: sign_out_everywhere_handler.New(sign_out_everywhere_usecase.New(store, clock)),
		// Publishes auth.v1.AccountDeleted, as the prune does for each guest it deletes.
		DeleteAccountHandler: delete_account_handler.New(delete_account_usecase.New(store, props.Events, clock)),
	}
	props.Runners.Add(prune_guests_usecase.NewRunner(config.Prune,
		log_prune_guests.New(prune_guests_usecase.New(config.Prune, store, props.Events, clock), props.Logger)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return authv1connect.NewAuthServiceHandler(authService, options...)
	}, authv1controller.NewRateLimitInterceptor(mintLimiter)); err != nil {
		return fmt.Errorf("failed to mount auth.v1: %w", err)
	}

	// The planet context verifies clicks with the public half of this signer, and
	// asks for it here rather than reading a key of its own. The seed never leaves
	// this module, and there is no second setting to keep in step with it.
	// The player module asks whether an account is linked before it gives it a username.
	internalService := authv1controller.InternalService{
		GetVerifyingKeyHandler: get_verifying_key_handler.New(signer),
		GetAccountHandler:      get_account_handler.New(get_account_usecase.New(store)),
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
		slog.Any("signIn", providers.Names()),
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

// Config is the `auth:` block. The minting half is the shared layer's; planet
// declares the verifying half of the same block and never sees the seed.
type Config struct {
	cpsession.SignerConfig `koanf:",squash"`

	// Per-IP throttle on minting, for both CreateSession paths together.
	RateLimiter cpratelimit.Config

	Turnstile turnstile.Config

	// Accounts and their sessions. Required when the module is on.
	Database cppg.Config

	Sessions accounts.Lifetime

	SignIn signin.Config

	// Each provider is offered while sign-in is on and its clientId is set. The secrets come from the environment.
	Google  signin.Client
	Discord signin.Client

	// Deletes the guests nobody has used for a long time.
	Prune prune_guests_usecase.Config
}

const defaultTurnstileAction = "session"

func (c Config) withDefaults() Config {
	if c.Turnstile.Action == "" {
		c.Turnstile.Action = defaultTurnstileAction
	}
	c.Sessions = c.Sessions.WithDefaults()
	// A guest is pruned no sooner than its cookie lapses.
	if c.Prune.IdleFor <= 0 {
		c.Prune.IdleFor = c.Sessions.GuestTTL
	}
	c.Prune = c.Prune.WithDefaults()
	return c
}

// Validate checks the whole block, the key pair included: planet reads the public half but does not check it.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	var databaseErr error
	if err := c.Database.Validate(); err != nil {
		databaseErr = fmt.Errorf("auth.database: %w", err)
	}
	return errors.Join(c.SignerConfig.Validate(), databaseErr, c.pruneError(), c.signInError())
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
