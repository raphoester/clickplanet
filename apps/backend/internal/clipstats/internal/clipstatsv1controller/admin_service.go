package clipstatsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clipstats/v1/clipstatsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/clipstatsv1controller/keep_snapshot_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/clipstatsv1controller/list_rows_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/clipstatsv1controller/list_snapshots_handler"
)

type AdminService struct {
	keep_snapshot_handler.KeepSnapshotHandler
	list_snapshots_handler.ListSnapshotsHandler
	list_rows_handler.ListRowsHandler
}

var _ clipstatsv1connect.AdminServiceHandler = AdminService{}
