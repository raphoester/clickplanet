package list_submissions_handler

import (
	"context"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
)

type Query interface {
	Kept(ctx context.Context) (*flagsv1.ListSubmissionsResponse, error)
}

func New(query Query) ListSubmissionsHandler {
	return ListSubmissionsHandler{query: query}
}

type ListSubmissionsHandler struct {
	query Query
}

func (h ListSubmissionsHandler) ListSubmissions(
	ctx context.Context,
	_ *connect.Request[flagsv1.ListSubmissionsRequest],
) (*connect.Response[flagsv1.ListSubmissionsResponse], error) {
	kept, err := h.query.Kept(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(kept), nil
}
