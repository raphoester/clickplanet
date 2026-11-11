package open_book_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_book_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, command open_book_usecase.Command) (bool, error)
}

func New(useCase UseCase) OpenBookHandler {
	return OpenBookHandler{useCase: useCase}
}

type OpenBookHandler struct {
	useCase UseCase
}

func (h OpenBookHandler) OpenBook(
	ctx context.Context,
	req *connect.Request[scoringv1.OpenBookRequest],
) (*connect.Response[scoringv1.OpenBookResponse], error) {
	book, err := books.ParseBookID(req.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := req.Msg.GetStartsAt().CheckValid(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("book %s has no start: %w", book, err))
	}
	_, err = h.useCase.Execute(ctx, open_book_usecase.Command{Book: book, StartsAt: req.Msg.GetStartsAt().AsTime()})
	if errors.Is(err, books.ErrRefused) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(&scoringv1.OpenBookResponse{}), nil
}
