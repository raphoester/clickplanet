package close_note_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type UseCase interface {
	Execute(ctx context.Context, note notes.NoteID) (time.Time, error)
}

func New(useCase UseCase) CloseNoteHandler {
	return CloseNoteHandler{useCase: useCase}
}

type CloseNoteHandler struct {
	useCase UseCase
}

func (h CloseNoteHandler) CloseNote(
	ctx context.Context,
	req *connect.Request[supportv1.CloseNoteRequest],
) (*connect.Response[supportv1.CloseNoteResponse], error) {
	note, err := notes.NoteIDOf(req.Msg.GetNoteId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidNoteID)
	}

	closedAt, err := h.useCase.Execute(ctx, note)
	switch {
	case errors.Is(err, notes.ErrUnknownNote):
		return nil, connect.NewError(connect.CodeNotFound, notes.ErrUnknownNote)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&supportv1.CloseNoteResponse{ClosedAt: timestamppb.New(closedAt)}), nil
}
