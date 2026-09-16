// Package auth wires who a caller is: accounts, and the sessions their cookies hold.
//
// No other module reads its tables or calls its code: they ask auth.v1.InternalService,
// which it serves on the internal listener only.
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/random_token_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resolve_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/resolve_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "auth"

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config, props)
		},
	}
}

func build(ctx context.Context, config Config, props cpbootstrap.Props) error {
	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect auth to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}
	props.Closers.Add("auth-postgres", db.Close)

	resolveAccount := resolve_account_usecase.New(
		postgres_account_store.New(db),
		uuid_id_provider.Provider{},
		random_token_generator.Generator{},
		config.Sessions,
		cptime.SystemClock{},
	)

	authService := authv1controller.AuthService{GetMeHandler: get_me_handler.New(resolveAccount)}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return authv1connect.NewAuthServiceHandler(authService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount the auth service: %w", err)
	}

	internalService := authv1controller.InternalService{ResolveAccountHandler: resolve_account_handler.New(resolveAccount)}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return authv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount the auth internal service: %w", err)
	}

	props.Logger.Info("auth built",
		slog.String("schema", config.Database.Schema),
		slog.Duration("guestTTL", config.Sessions.WithDefaults().GuestTTL),
	)

	return nil
}

type Config struct {
	// Off registers nothing: auth.v1 404s, and session.accounts must be off too.
	Enabled bool

	Database cppg.Config

	Sessions accounts.Lifetime
}

// Validate refuses only a missing database, and only when the module is on.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("auth.database: %w", err)
	}
	return nil
}
