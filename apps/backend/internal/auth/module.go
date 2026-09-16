// Package auth wires who a caller is and what it has to prove before it may
// click: Turnstile, the account its cookie holds, and the click token it mints.
//
// The planet context verifies that token from the same `auth:` block, and
// builds its own signer from it rather than being handed this one.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/random_token_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/create_anonymous_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/create_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/get_me_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/open_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/turnstile"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/turnstile_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/create_session_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/sessionv1controller"
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
			return build(ctx, config.withDefaults(), props)
		},
	}
}

func build(ctx context.Context, config Config, props cpbootstrap.Props) error {
	signer, err := cpsession.NewSigner(config.Config)
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

	authService := authv1controller.AuthService{
		CreateSessionHandler: create_session_handler.New(
			create_session_usecase.New(attester, store, uuid_id_provider.Provider{}, random_token_generator.Generator{},
				signer, config.Sessions, clock),
			props.Logger,
		),
		GetMeHandler: get_me_handler.New(get_me_usecase.New(store, clock)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return authv1connect.NewAuthServiceHandler(authService, options...)
	}, authv1controller.NewRateLimitInterceptor(mintLimiter)); err != nil {
		return fmt.Errorf("failed to mount auth.v1: %w", err)
	}

	sessionService := sessionv1controller.NewSessionService(
		create_anonymous_session_usecase.New(attester, signer, clock), props.Logger)
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return sessionv1connect.NewSessionServiceHandler(sessionService, options...)
	}, sessionv1controller.NewRateLimitInterceptor(mintLimiter)); err != nil {
		return fmt.Errorf("failed to mount the deprecated session.v1: %w", err)
	}

	props.Logger.Info("auth built",
		slog.String("schema", config.Database.Schema),
		slog.Duration("ttl", config.TTL),
		slog.Bool("enforce", config.Enforce),
		slog.Bool("turnstile", config.Turnstile.Enabled),
		slog.Duration("guestTTL", config.Sessions.GuestTTL),
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

// Config is the `auth:` block. The token half is the shared layer's, because the
// planet context declares the same type to verify what this one mints.
type Config struct {
	cpsession.Config `koanf:",squash"`

	// Per-IP throttle on minting, for both CreateSession paths together.
	RateLimiter cpratelimit.Config

	Turnstile turnstile.Config

	// Accounts and their sessions. Required when the module is on.
	Database cppg.Config

	Sessions accounts.Lifetime
}

const defaultTurnstileAction = "session"

func (c Config) withDefaults() Config {
	if c.Turnstile.Action == "" {
		c.Turnstile.Action = defaultTurnstileAction
	}
	c.Sessions = c.Sessions.WithDefaults()
	return c
}

// Validate checks the whole block, the token half included: planet reads it but does not check it.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	var databaseErr error
	if err := c.Database.Validate(); err != nil {
		databaseErr = fmt.Errorf("auth.database: %w", err)
	}
	return errors.Join(c.Config.Validate(), databaseErr)
}
