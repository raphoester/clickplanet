package list_reports_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
)

type Query interface {
	Reports(ctx context.Context, since time.Time, limit uint32) (*supportv1.ListReportsResponse, error)
}

func New(query Query) ListReportsHandler {
	return ListReportsHandler{query: query}
}

type ListReportsHandler struct {
	query Query
}

func (h ListReportsHandler) ListReports(
	ctx context.Context,
	req *connect.Request[supportv1.ListReportsRequest],
) (*connect.Response[supportv1.ListReportsResponse], error) {
	var since time.Time
	if asked := req.Msg.GetSince(); asked != nil {
		since = asked.AsTime()
	}

	reports, err := h.query.Reports(ctx, since, req.Msg.GetLimit())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(reports), nil
}
