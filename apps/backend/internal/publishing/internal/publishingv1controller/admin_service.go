package publishingv1controller

import (
	"context"

	"connectrpc.com/connect"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/create_media_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/get_post_links_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/get_room_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_posts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_rows_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_snapshots_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/plan_post_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/schedule_post_handler"
)

type AdminService struct {
	create_media_upload_handler.CreateMediaUploadHandler
	get_room_handler.GetRoomHandler
	plan_post_handler.PlanPostHandler
	schedule_post_handler.SchedulePostHandler
	list_posts_handler.ListPostsHandler
	StatsReader
}

// The operator's calls and another module's: the same handlers, one value mounted on both listeners.
var (
	_ publishingv1connect.AdminServiceHandler    = AdminService{}
	_ publishingv1connect.InternalServiceHandler = AdminService{}
)

// The posts' numbers, and the links the snapshots read: Stats, or StatsOff with no database.
type StatsReader interface {
	ListSnapshots(context.Context, *connect.Request[publishingv1.ListSnapshotsRequest]) (*connect.Response[publishingv1.ListSnapshotsResponse], error)
	ListRows(context.Context, *connect.Request[publishingv1.ListRowsRequest]) (*connect.Response[publishingv1.ListRowsResponse], error)
	GetPostLinks(context.Context, *connect.Request[publishingv1.GetPostLinksRequest]) (*connect.Response[publishingv1.GetPostLinksResponse], error)
}

type Stats struct {
	list_snapshots_handler.ListSnapshotsHandler
	list_rows_handler.ListRowsHandler
	get_post_links_handler.GetPostLinksHandler
}

var _ StatsReader = Stats{}
