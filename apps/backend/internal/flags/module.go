package flags

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1/flagsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/restate_flags_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_flags_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_flags_handler/flags_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const moduleName = "flags"

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
	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the flags module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the flags module to redis: %w", err)
	}

	store := postgres_flag_store.New(db)
	// Each module keeps what it heard: one that subscribed after a flag was told hears it at the next boot.
	if err := restate_flags_usecase.New(db, store, cpoutbox.New(db)).Execute(ctx); err != nil {
		_ = streams.Close()
		_ = db.Close()
		return err //nolint:wrapcheck // the use case named it.
	}

	// Not a closer: closers run before the runners stop, and the relay uses the pool.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger,
		cpredis.CloseAfter(streams, props.Logger, cpoutbox.NewRelay("flags-outbox", db, streams, props.Logger))))

	list := list_flags_handler.New(flags_query.NewPostgresQuery(db))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return flagsv1connect.NewFlagServiceHandler(flagsv1controller.FlagService{ListFlagsHandler: list}, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount flags.v1.FlagService: %w", err)
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return flagsv1connect.NewAdminServiceHandler(flagsv1controller.AdminService{ListFlagsHandler: list}, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount flags.v1.AdminService: %w", err)
	}

	props.Logger.Info("flags built", slog.String("schema", config.Database.Schema))
	return nil
}

type Config struct {
	// The flags opened since the game shipped, and the outbox of what is told of them.
	Database cppg.Config
	// Where it tells each flag opened or closed.
	Redis cpredis.Config
}

func (c Config) Validate() error {
	var errs []error
	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("flags.database: %w", err))
	}
	if err := c.Redis.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("flags.redis: %w", err))
	}
	return errors.Join(errs...)
}
