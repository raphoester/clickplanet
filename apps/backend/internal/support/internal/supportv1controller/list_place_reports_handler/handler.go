package list_place_reports_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
)

type Query interface {
	PlaceReports(ctx context.Context, since time.Time, limit uint32, openOnly bool) (*supportv1.ListPlaceReportsResponse, error)
}

func New(query Query) ListPlaceReportsHandler {
	return ListPlaceReportsHandler{query: query}
}

type ListPlaceReportsHandler struct {
	query Query
}

func (h ListPlaceReportsHandler) ListPlaceReports(
	ctx context.Context,
	req *connect.Request[supportv1.ListPlaceReportsRequest],
) (*connect.Response[supportv1.ListPlaceReportsResponse], error) {
	var since time.Time
	if asked := req.Msg.GetSince(); asked != nil {
		since = asked.AsTime()
	}

	reports, err := h.query.PlaceReports(ctx, since, req.Msg.GetLimit(), req.Msg.GetOpenOnly())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(reports), nil
}
