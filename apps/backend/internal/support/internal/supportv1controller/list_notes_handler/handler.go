package list_notes_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_notes_handler/notes_query"
)

type Query interface {
	Notes(ctx context.Context, since time.Time, kind supportv1.NoteKind, limit uint32) (*supportv1.ListNotesResponse, error)
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

	notes, err := h.query.Notes(ctx, since, req.Msg.GetKind(), req.Msg.GetLimit())
	switch {
	case errors.Is(err, notes_query.ErrInvalidKind):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(notes), nil
}
