package answer_note_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/usecases/answer_note_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type UseCase interface {
	Execute(ctx context.Context, in answer_note_usecase.In) (time.Time, error)
}

func New(useCase UseCase) AnswerNoteHandler {
	return AnswerNoteHandler{useCase: useCase}
}

type AnswerNoteHandler struct {
	useCase UseCase
}

func (h AnswerNoteHandler) AnswerNote(
	ctx context.Context,
	req *connect.Request[supportv1.AnswerNoteRequest],
) (*connect.Response[supportv1.AnswerNoteResponse], error) {
	note, err := notes.NoteIDOf(req.Msg.GetNoteId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidNoteID)
	}

	sentAt, err := h.useCase.Execute(ctx, answer_note_usecase.In{Note: note, Text: req.Msg.GetText()})
	switch {
	case errors.Is(err, notes.ErrUnknownNote):
		return nil, connect.NewError(connect.CodeNotFound, notes.ErrUnknownNote)
	case errors.Is(err, notes.ErrNoReplyAddress):
		return nil, connect.NewError(connect.CodeFailedPrecondition, notes.ErrNoReplyAddress)
	case errors.Is(err, notes.ErrInvalidText):
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidText)
	case errors.Is(err, notes.ErrNotSent):
		return nil, connect.NewError(connect.CodeUnavailable, notes.ErrNotSent)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&supportv1.AnswerNoteResponse{SentAt: timestamppb.New(sentAt)}), nil
}
