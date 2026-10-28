package clipsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1/clipsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_appearance_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_snapshot_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_rows_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_snapshots_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/make_cast_handler"
)

type AdminService struct {
	keep_snapshot_handler.KeepSnapshotHandler
	list_snapshots_handler.ListSnapshotsHandler
	list_rows_handler.ListRowsHandler
	make_cast_handler.MakeCastHandler
	keep_appearance_handler.KeepAppearanceHandler
}

var _ clipsv1connect.AdminServiceHandler = AdminService{}
