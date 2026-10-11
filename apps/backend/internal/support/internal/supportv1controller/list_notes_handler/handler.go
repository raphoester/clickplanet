package list_notes_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
)

type Query interface {
	Notes(ctx context.Context, since time.Time, limit uint32) (*supportv1.ListNotesResponse, error)
}

func New(query Query) ListNotesHandler {
	return ListNotesHandler{query: query}
}

type ListNotesHandler struct {
	query Query
}

func (h ListNotesHandler) ListNotes(
	ctx context.Context,
	req *connect.Request[supportv1.ListNotesRequest],
) (*connect.Response[supportv1.ListNotesResponse], error) {
	var since time.Time
	if asked := req.Msg.GetSince(); asked != nil {
		since = asked.AsTime()
	}

	notes, err := h.query.Notes(ctx, since, req.Msg.GetLimit())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(notes), nil
}
