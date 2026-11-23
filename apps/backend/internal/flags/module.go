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
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/tell_streams_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_flags_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_flags_handler/flags_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/subscribers/told_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "flags"

// The flags a page that reads nothing may fall behind before its stream is closed: it opens it again and reads the list.
const streamBuffer = 32

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

	redis := cpredis.New(config.Redis, props.Metrics)
	if err := redis.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the flags module to redis: %w", err)
	}

	store := postgres_flag_store.New(db)
	// Facts decided before this boot are told again for the other modules, never to the pages: they read the list.
	streams := feed.NewBoard(streamBuffer)
	tell := tell_streams_usecase.New(feed.NewsSince(cptime.SystemClock{}.Now()), store, streams)
	opened, err := cpredis.Subscribe(ctx, redis, "flags-streams-opened",
		log_subscriber.New(told_subscriber.NewOpened(tell), props.Logger))
	if err != nil {
		_ = redis.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the streams to flags.v1.FlagOpened: %w", err)
	}
	closed, err := cpredis.Subscribe(ctx, redis, "flags-streams-closed",
		log_subscriber.New(told_subscriber.NewClosed(tell), props.Logger))
	if err != nil {
		_ = redis.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the streams to flags.v1.FlagClosed: %w", err)
	}

	// Each module keeps what it heard: one that subscribed after a flag was told hears it at the next boot.
	if err := restate_flags_usecase.New(db, store, cpoutbox.New(db)).Execute(ctx); err != nil {
		_ = redis.Close()
		_ = db.Close()
		return err //nolint:wrapcheck // the use case named it.
	}

	// Not a closer: closers run before the runners stop, and the relay uses the pool.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger,
		cpredis.CloseAfter(redis, props.Logger, opened, closed, cpoutbox.NewRelay("flags-outbox", db, redis, props.Logger))))

	list := list_flags_handler.New(flags_query.NewPostgresQuery(db))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return flagsv1connect.NewFlagServiceHandler(flagsv1controller.FlagService{
			ListFlagsHandler:       list,
			ListenForEventsHandler: listen_for_events_handler.New(listen_for_events_usecase.New(streams, props.Server.StreamHeartbeat)),
		}, options...)
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
