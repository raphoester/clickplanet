package clips

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1/clipsv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_snapshot_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_rows_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_rows_handler/rows_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_snapshots_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_snapshots_handler/snapshots_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots/postgres_snapshot_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots/usecases/keep_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const moduleName = "clips"

// The server's side of the clips job: the numbers of the clips it posted, a snapshot a day, which `clips stats` sends
// over ssh and the backoffice reads. An AdminService alone: nothing of it is on the public router.
func NewModule(config ModuleConfig) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config, props)
		},
	}
}

func build(ctx context.Context, config ModuleConfig, props cpbootstrap.Props) error {
	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the clips module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}
	// A closer, not cppg.CloseAfter: the module runs nothing in the background, and CloseAfter of no runner would close
	// the pool as soon as the server starts.
	props.Closers.Add("clips-postgres", db.Close)

	adminService := clipsv1controller.AdminService{
		KeepSnapshotHandler:  keep_snapshot_handler.New(keep_snapshot_usecase.New(postgres_snapshot_store.New(db))),
		ListSnapshotsHandler: list_snapshots_handler.New(snapshots_query.NewPostgresQuery(db)),
		ListRowsHandler:      list_rows_handler.New(rows_query.NewPostgresQuery(db)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return clipsv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount clips.v1.AdminService: %w", err)
	}

	props.Logger.Info("clips built", slog.String("schema", config.Database.Schema))
	return nil
}

// The server's settings, beside the job's Config: the job runs in a workflow, the module in the server.
type ModuleConfig struct {
	Database cppg.Config
}

func (c ModuleConfig) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("clips.database: %w", err)
	}
	return nil
}
