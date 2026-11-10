package publishingv1controller

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
)

var (
	ErrOff      = errors.New("publishing is off: the server holds no key of the posting service")
	ErrNoNumber = errors.New("the posts' numbers are off: publishing has no database")
)

// The service while the server holds no key: every call is FailedPrecondition.
type Off struct{}

var (
	_ publishingv1connect.AdminServiceHandler    = Off{}
	_ publishingv1connect.InternalServiceHandler = Off{}
)

func (Off) CreateMediaUpload(
	context.Context, *connect.Request[publishingv1.CreateMediaUploadRequest],
) (*connect.Response[publishingv1.CreateMediaUploadResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrOff)
}

func (Off) GetRoom(context.Context, *connect.Request[publishingv1.GetRoomRequest]) (*connect.Response[publishingv1.GetRoomResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrOff)
}

func (Off) PlanPost(context.Context, *connect.Request[publishingv1.PlanPostRequest]) (*connect.Response[publishingv1.PlanPostResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrOff)
}

func (Off) SchedulePost(
	context.Context, *connect.Request[publishingv1.SchedulePostRequest],
) (*connect.Response[publishingv1.SchedulePostResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrOff)
}

func (Off) ListPosts(context.Context, *connect.Request[publishingv1.ListPostsRequest]) (*connect.Response[publishingv1.ListPostsResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrOff)
}

func (Off) ListSnapshots(
	context.Context, *connect.Request[publishingv1.ListSnapshotsRequest],
) (*connect.Response[publishingv1.ListSnapshotsResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrOff)
}

func (Off) ListRows(context.Context, *connect.Request[publishingv1.ListRowsRequest]) (*connect.Response[publishingv1.ListRowsResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrOff)
}

// The posts' numbers with no database: posting still works, and no snapshot is taken.
type StatsOff struct{}

var _ StatsReader = StatsOff{}

func (StatsOff) ListSnapshots(
	context.Context, *connect.Request[publishingv1.ListSnapshotsRequest],
) (*connect.Response[publishingv1.ListSnapshotsResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrNoNumber)
}

func (StatsOff) ListRows(context.Context, *connect.Request[publishingv1.ListRowsRequest]) (*connect.Response[publishingv1.ListRowsResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrNoNumber)
}

func (Off) GetPostLinks(
	context.Context, *connect.Request[publishingv1.GetPostLinksRequest],
) (*connect.Response[publishingv1.GetPostLinksResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrOff)
}

// No snapshot read any link.
func (StatsOff) GetPostLinks(
	context.Context, *connect.Request[publishingv1.GetPostLinksRequest],
) (*connect.Response[publishingv1.GetPostLinksResponse], error) {
	return connect.NewResponse(&publishingv1.GetPostLinksResponse{}), nil
}
