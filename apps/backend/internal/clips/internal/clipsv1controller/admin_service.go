package clipsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1/clipsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_snapshot_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_rows_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_snapshots_handler"
)

type AdminService struct {
	keep_snapshot_handler.KeepSnapshotHandler
	list_snapshots_handler.ListSnapshotsHandler
	list_rows_handler.ListRowsHandler
}

var _ clipsv1connect.AdminServiceHandler = AdminService{}
