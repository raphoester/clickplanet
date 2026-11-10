package list_snapshots_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
)

type Query interface {
	Snapshots(ctx context.Context, since, until time.Time, limit uint32) (*publishingv1.ListSnapshotsResponse, error)
}

func New(query Query) ListSnapshotsHandler {
	return ListSnapshotsHandler{query: query}
}

type ListSnapshotsHandler struct {
	query Query
}

func (h ListSnapshotsHandler) ListSnapshots(
	ctx context.Context,
	req *connect.Request[publishingv1.ListSnapshotsRequest],
) (*connect.Response[publishingv1.ListSnapshotsResponse], error) {
	var since, until time.Time
	if asked := req.Msg.GetSince(); asked != nil {
		since = asked.AsTime()
	}
	if asked := req.Msg.GetUntil(); asked != nil {
		until = asked.AsTime()
	}

	listed, err := h.query.Snapshots(ctx, since, until, req.Msg.GetLimit())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(listed), nil
}
