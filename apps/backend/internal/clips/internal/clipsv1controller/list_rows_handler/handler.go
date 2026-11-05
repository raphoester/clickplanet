package list_rows_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_rows_handler/rows_query"
)

type Query interface {
	Rows(ctx context.Context, since, until time.Time, limit uint32, page string) (*clipsv1.ListRowsResponse, error)
}

func New(query Query) ListRowsHandler {
	return ListRowsHandler{query: query}
}

type ListRowsHandler struct {
	query Query
}

func (h ListRowsHandler) ListRows(
	ctx context.Context,
	req *connect.Request[clipsv1.ListRowsRequest],
) (*connect.Response[clipsv1.ListRowsResponse], error) {
	var since, until time.Time
	if asked := req.Msg.GetSince(); asked != nil {
		since = asked.AsTime()
	}
	if asked := req.Msg.GetUntil(); asked != nil {
		until = asked.AsTime()
	}

	listed, err := h.query.Rows(ctx, since, until, req.Msg.GetLimit(), req.Msg.GetPage())
	switch {
	case errors.Is(err, rows_query.ErrInvalidPage):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(listed), nil
}
