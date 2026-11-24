package write_letter_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters/usecases/write_letter_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/lettermessage"
)

type UseCase interface {
	Execute(ctx context.Context, in write_letter_usecase.In) (letters.Snapshot, error)
}

func New(useCase UseCase) WriteLetterHandler {
	return WriteLetterHandler{useCase: useCase}
}

type WriteLetterHandler struct {
	useCase UseCase
}

func (h WriteLetterHandler) WriteLetter(
	ctx context.Context,
	req *connect.Request[supportv1.WriteLetterRequest],
) (*connect.Response[supportv1.WriteLetterResponse], error) {
	letter, err := h.useCase.Execute(ctx, write_letter_usecase.In{
		Letter: req.Msg.GetLetterId(), Account: req.Msg.GetAccountId(), Subject: req.Msg.GetSubject(),
		Text: req.Msg.GetText(),
	})
	switch {
	case errors.Is(err, letters.ErrUnknownLetter):
		return nil, connect.NewError(connect.CodeNotFound, letters.ErrUnknownLetter)
	case errors.Is(err, notes.ErrNoReplyAddress):
		return nil, connect.NewError(connect.CodeFailedPrecondition, notes.ErrNoReplyAddress)
	case errors.Is(err, notes.ErrInvalidAccount), errors.Is(err, letters.ErrInvalidSubject),
		errors.Is(err, notes.ErrInvalidText):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, notes.ErrNotSent):
		return nil, connect.NewError(connect.CodeUnavailable, notes.ErrNotSent)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&supportv1.WriteLetterResponse{Letter: lettermessage.Letter(letter)}), nil
}
