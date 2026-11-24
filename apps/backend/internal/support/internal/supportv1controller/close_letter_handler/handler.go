package close_letter_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
)

type UseCase interface {
	Execute(ctx context.Context, letter string) (time.Time, error)
}

func New(useCase UseCase) CloseLetterHandler {
	return CloseLetterHandler{useCase: useCase}
}

type CloseLetterHandler struct {
	useCase UseCase
}

func (h CloseLetterHandler) CloseLetter(
	ctx context.Context,
	req *connect.Request[supportv1.CloseLetterRequest],
) (*connect.Response[supportv1.CloseLetterResponse], error) {
	closedAt, err := h.useCase.Execute(ctx, req.Msg.GetLetterId())
	switch {
	case errors.Is(err, letters.ErrUnknownLetter):
		return nil, connect.NewError(connect.CodeNotFound, letters.ErrUnknownLetter)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&supportv1.CloseLetterResponse{ClosedAt: timestamppb.New(closedAt)}), nil
}
