package clipstats

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clipstats/v1/clipstatsv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/clipstatsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/clipstatsv1controller/keep_snapshot_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/clipstatsv1controller/list_rows_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/clipstatsv1controller/list_rows_handler/rows_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/clipstatsv1controller/list_snapshots_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/clipstatsv1controller/list_snapshots_handler/snapshots_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/snapshots/postgres_snapshot_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/snapshots/usecases/keep_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const moduleName = "clipstats"

// The numbers of the clips the clips job posted, a snapshot a day, which the job sends over ssh and the backoffice
// reads. An AdminService alone: nothing of it is on the public router.
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
		return fmt.Errorf("failed to connect the clipstats module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}
	// A closer, not cppg.CloseAfter: the module runs nothing in the background, and CloseAfter of no runner would close
	// the pool as soon as the server starts.
	props.Closers.Add("clipstats-postgres", db.Close)

	adminService := clipstatsv1controller.AdminService{
		KeepSnapshotHandler:  keep_snapshot_handler.New(keep_snapshot_usecase.New(postgres_snapshot_store.New(db))),
		ListSnapshotsHandler: list_snapshots_handler.New(snapshots_query.NewPostgresQuery(db)),
		ListRowsHandler:      list_rows_handler.New(rows_query.NewPostgresQuery(db)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return clipstatsv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount clipstats.v1.AdminService: %w", err)
	}

	props.Logger.Info("clipstats built", slog.String("schema", config.Database.Schema))
	return nil
}

type Config struct {
	Database cppg.Config
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("clipstats.database: %w", err)
	}
	return nil
}
